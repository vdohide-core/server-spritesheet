package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"server-spritesheet/internal/db/models"
	"server-spritesheet/internal/utils"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
)

// ─── Error Categorization ────────────────────────────────────

func categorizeError(errMsg string) string {
	e := strings.ToLower(errMsg)
	switch {
	case strings.Contains(e, "codec") || strings.Contains(e, "encode"):
		return "codec"
	case strings.Contains(e, "ffmpeg") || strings.Contains(e, "ffprobe"):
		return "ffmpeg"
	case strings.Contains(e, "upload") || strings.Contains(e, "s3"):
		return "upload"
	case strings.Contains(e, "download") || strings.Contains(e, "timeout") || strings.Contains(e, "connection"):
		return "network"
	case strings.Contains(e, "probe"):
		return "probe"
	case strings.Contains(e, "thumbnail") || strings.Contains(e, "sprite"):
		return "thumbnail"
	default:
		return "unknown"
	}
}

// ─── isCancelled ─────────────────────────────────────────────

func isCancelled(ctx context.Context, processID string) bool {
	p, err := models.VideoProcessModel.FindByID(ctx, processID)
	if err != nil {
		// Record missing ≠ cancelled — another service may have deleted it
		// Continue processing rather than silently aborting
		log.Printf("⚠️  isCancelled: FindByID(%s) error: %v — NOT treating as cancelled", processID, err)
		return false
	}
	status := derefStr(p.Status)
	if status == models.ProcessStatusCancelled {
		log.Printf("⚠️  isCancelled: process %s has status=cancelled", processID)
		return true
	}
	return false
}

// ─── failProcess ─────────────────────────────────────────────
// Marks process as failed. After 3 retries, keeps failed process as log.

func failProcess(ctx context.Context, processID, slug, errMsg string) {
	utils.LogMain("❌ [%s] ERROR: %s", slug, errMsg)
	category := categorizeError(errMsg)

	// Read current retryCount → increment manually
	retryNum := 1
	current, _ := models.VideoProcessModel.FindByID(ctx, processID)
	if current != nil && current.RetryCount != nil {
		retryNum = *current.RetryCount + 1
	}
	log.Printf("🔍 [%s] failProcess: processID=%s, currentRetry=%v, newRetry=%d", slug, processID, current != nil && current.RetryCount != nil, retryNum)

	// Use raw MongoDB UpdateOne to avoid goose auto-wrapping $set
	result, err := models.VideoProcessModel.Col().UpdateOne(ctx,
		bson.M{"_id": processID},
		bson.M{"$set": bson.M{
			"status":        models.ProcessStatusFailed,
			"error":         errMsg,
			"errorCategory": category,
			"retryCount":    retryNum,
			"updatedAt":     time.Now(),
		}},
	)

	if err != nil {
		log.Printf("❌ [%s] Process update failed: %v", slug, err)
		return
	}

	log.Printf("🔍 [%s] failProcess: matched=%d, modified=%d", slug, result.MatchedCount, result.ModifiedCount)

	// Verify readback
	verify, _ := models.VideoProcessModel.FindByID(ctx, processID)
	if verify != nil {
		vRetry := 0
		if verify.RetryCount != nil {
			vRetry = *verify.RetryCount
		}
		log.Printf("🔍 [%s] failProcess verify: status=%s, retryCount=%d", slug, derefStr(verify.Status), vRetry)
	} else {
		log.Printf("⚠️  [%s] failProcess verify: process NOT FOUND after update!", slug)
	}

	log.Printf("❌ [%s] Failed: %s [%s] (retry %d/3)", slug, errMsg, category, retryNum)

	// Diagnostic: delayed re-check — see if record disappears while we wait
	time.Sleep(2 * time.Second)
	recheck, _ := models.VideoProcessModel.FindByID(ctx, processID)
	if recheck == nil {
		log.Printf("🚨 [%s] CRITICAL: process %s VANISHED within 2 seconds of failProcess!", slug, processID)
		// Check if ANY documents exist
		allDocs, _ := models.VideoProcessModel.Find(ctx, bson.M{})
		log.Printf("🚨 [%s] Collection has %d documents after vanish", slug, len(allDocs))
	} else {
		log.Printf("✅ [%s] Process %s still exists after 2s delay", slug, processID)
	}
}

// ─── Progress Helpers ────────────────────────────────────────

func updateTimelineStep(ctx context.Context, processID, step, status string, percent float64) {
	models.VideoProcessModel.UpdateByID(ctx, processID, bson.M{"$set": bson.M{
		fmt.Sprintf("timeline.%s.status", step):  status,
		fmt.Sprintf("timeline.%s.percent", step): percent,
		"updatedAt":                              time.Now(),
	}})
}

func startStep(ctx context.Context, processID, step string) {
	now := time.Now()
	models.VideoProcessModel.UpdateByID(ctx, processID, bson.M{"$set": bson.M{
		fmt.Sprintf("timeline.%s.status", step):    models.StepStatusProcessing,
		fmt.Sprintf("timeline.%s.percent", step):   0,
		fmt.Sprintf("timeline.%s.startedAt", step): now,
		"updatedAt": now,
	}})
}

func completeStep(ctx context.Context, processID, step string) {
	now := time.Now()
	models.VideoProcessModel.UpdateByID(ctx, processID, bson.M{"$set": bson.M{
		fmt.Sprintf("timeline.%s.status", step):  models.StepStatusCompleted,
		fmt.Sprintf("timeline.%s.percent", step): 100,
		fmt.Sprintf("timeline.%s.endedAt", step): now,
		"updatedAt":                              now,
	}})
}

func updateOverallPercent(ctx context.Context, processID string, percent float64) {
	models.VideoProcessModel.UpdateByID(ctx, processID, bson.M{"$set": bson.M{
		"overallPercent": percent,
		"updatedAt":      time.Now(),
	}})
}

// ─── Clone media to cloned files ─────────────────────────────

func cloneMediaToClonedFiles(ctx context.Context, sourceFileID string, media models.Media, slug string) {
	cursor, err := models.FileModel.FindRaw(ctx, bson.M{
		"clonedFrom":         sourceFileID,
		"type":               models.FileTypeVideo,
		"metadata.trashedAt": bson.M{"$exists": false},
		"metadata.deletedAt": bson.M{"$exists": false},
	})
	if err != nil {
		return
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var clonedFile models.File
		if err := cursor.Decode(&clonedFile); err != nil {
			continue
		}

		filter := bson.M{"fileId": clonedFile.ID, "type": media.Type}
		if media.Resolution != nil {
			filter["resolution"] = *media.Resolution
		}
		existCount, _ := models.MediaModel.CountDocuments(ctx, filter)
		if existCount > 0 {
			continue
		}

		now := time.Now()
		slug11 := utils.RandomString(11, true)
		clonedMedia := models.Media{
			ID:         uuid.New().String(),
			Type:       media.Type,
			FileName:   media.FileName,
			MimeType:   media.MimeType,
			Resolution: media.Resolution,
			StorageID:  media.StorageID,
			Slug:       slug11,
			FileID:     &clonedFile.ID,
			Metadata:   media.Metadata,
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		clonedFrom := sourceFileID
		clonedMedia.ClonedFrom = &clonedFrom

		if _, err := models.MediaModel.Create(ctx, &clonedMedia); err != nil {
			log.Printf("⚠️  [%s] Failed to clone media to %s: %v", slug, clonedFile.ID, err)
			continue
		}
		log.Printf("📋 [%s] Cloned media → file %s", slug, clonedFile.ID)
	}
}
