package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"server-spritesheet/internal/archive"
	"server-spritesheet/internal/config"
	"server-spritesheet/internal/db/models"
	"server-spritesheet/internal/downloader"
	"server-spritesheet/internal/spritesheet"
	"server-spritesheet/internal/uploader"
	"server-spritesheet/internal/utils"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func newUUID() string { return uuid.New().String() }

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func hasThumbnailMedia(ctx context.Context, fileID string) bool {
	count, _ := models.MediaModel.CountDocuments(ctx, bson.M{
		"fileId":    fileID,
		"type":      models.MediaTypeThumbnail,
		"deletedAt": bson.M{"$exists": false},
	})
	return count > 0
}

// hasPendingSpriteIngest — sprite.zip uploaded, waiting for server-transfer.
func hasPendingSpriteIngest(ctx context.Context, fileID string) bool {
	count, _ := models.IngestModel.CountDocuments(ctx, bson.M{
		"fileId":     fileID,
		"fileName":   models.SpriteZipName,
		"sourceType": models.IngestSourceTypeProcessed,
		"deletedAt":  bson.M{"$exists": false},
	})
	return count > 0
}

func createSpriteIngest(ctx context.Context, fileID string, s3Storage *models.Storage, zipSize int64) error {
	if hasPendingSpriteIngest(ctx, fileID) {
		return nil
	}
	now := time.Now()
	ingestPath := fileID + "/" + models.SpriteZipName
	storageID := s3Storage.ID
	ingest := models.Ingest{
		ID:         newUUID(),
		FileID:     &fileID,
		StorageID:  &storageID,
		FileName:   models.SpriteZipName,
		Status:     "completed",
		Size:       zipSize,
		Path:       &ingestPath,
		SourceType: models.IngestSourceTypeProcessed,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	_, err := models.IngestModel.Create(ctx, &ingest)
	if err != nil {
		return err
	}
	log.Printf("✅ Created ingest: fileId=%s fileName=%s path=%s storageId=%s", fileID, models.SpriteZipName, ingestPath, storageID)
	return nil
}

// repairOrphanSpriteOnS3 creates ingest when sprite.zip is on S3 but ingest was missing.
func repairOrphanSpriteOnS3(ctx context.Context, fileID, slug string) bool {
	s3Storage, err := resolveS3TempStorage(ctx)
	if err != nil {
		return false
	}
	key := fileID + "/" + models.SpriteZipName
	exists, err := downloader.ObjectExists(s3Storage, key)
	if err != nil || !exists {
		return false
	}
	log.Printf("🔧 [%s] sprite.zip on S3 without ingest — creating ingest", slug)
	if err := createSpriteIngest(ctx, fileID, s3Storage, 0); err != nil {
		log.Printf("⚠️  [%s] Failed to create orphan ingest: %v", slug, err)
		return false
	}
	return true
}

func hasSourceVideo(ctx context.Context, fileID string) bool {
	count, _ := models.MediaModel.CountDocuments(ctx, bson.M{
		"fileId": fileID,
		"type":   models.MediaTypeVideo,
		"resolution": bson.M{"$in": []string{
			models.ResolutionOriginal,
			models.Resolution360, models.Resolution480,
			models.Resolution720, models.Resolution1080,
		}},
		"deletedAt": bson.M{"$exists": false},
	})
	return count > 0
}

func isFileExcluded(fileID string, excludeIDs map[string]bool) bool {
	return excludeIDs[fileID]
}

// spritesheetBlockReason returns why a file cannot be claimed (empty = ok).
func spritesheetBlockReason(ctx context.Context, file *models.File, excludeIDs map[string]bool) string {
	if isFileExcluded(file.ID, excludeIDs) {
		return "file_errors"
	}
	if !hasSourceVideo(ctx, file.ID) {
		return "no_video_media"
	}
	if hasThumbnailMedia(ctx, file.ID) {
		return "has_thumbnail"
	}
	if hasPendingSpriteIngest(ctx, file.ID) {
		return "pending_sprite_ingest"
	}

	existing, err := models.VideoProcessModel.FindOne(ctx, bson.M{
		"fileId": file.ID, "processType": models.ProcessTypeSpritesheet,
	})
	if err != nil {
		return ""
	}

	status := derefStr(existing.Status)
	owner := derefStr(existing.WorkerID)
	if status == models.ProcessStatusProcessing {
		if owner == workerID {
			return "own_processing_resume"
		}
		return "processing_by_" + owner
	}
	if status == models.ProcessStatusFailed {
		if owner == workerID {
			return "own_failed_retry"
		}
		return "failed_by_" + owner
	}
	return "active_process"
}

func clearStaleSpritesheetProcess(ctx context.Context, fileID, slug string) {
	existing, err := models.VideoProcessModel.FindOne(ctx, bson.M{
		"fileId": fileID, "processType": models.ProcessTypeSpritesheet,
	})
	if err != nil {
		return
	}

	status := derefStr(existing.Status)
	owner := derefStr(existing.WorkerID)

	if status == models.ProcessStatusProcessing && owner != workerID {
		w, wErr := models.WorkerModel.FindOne(ctx, bson.M{"workerId": owner})
		stale := wErr != nil
		if !stale && !w.HeartbeatAt.IsZero() {
			stale = time.Since(w.HeartbeatAt) > 5*time.Minute
		}
		if stale {
			log.Printf("🧹 [%s] Removing stale processing lock (worker %s offline)", slug, owner)
			models.VideoProcessModel.DeleteByID(ctx, existing.ID)
		}
		return
	}

	if status == models.ProcessStatusFailed && owner != workerID {
		retry := 0
		if existing.RetryCount != nil {
			retry = *existing.RetryCount
		}
		if retry >= 3 || time.Since(existing.UpdatedAt) > 30*time.Minute {
			log.Printf("🧹 [%s] Removing stale failed process (worker %s)", slug, owner)
			models.VideoProcessModel.DeleteByID(ctx, existing.ID)
		}
	}
}

// logIdleDiagnostics explains why no job was claimed (logged periodically).
func logIdleDiagnostics(ctx context.Context) {
	excludeIDs := map[string]bool{}
	errorDocs, _ := models.FileErrorModel.Find(ctx, bson.M{"errorType": "spritesheet"})
	for _, e := range errorDocs {
		excludeIDs[e.FileID] = true
	}

	readyCount, _ := models.FileModel.CountDocuments(ctx, bson.M{
		"status": models.FileStatusReady, "type": models.FileTypeVideo,
		"clonedFrom": bson.M{"$exists": false},
		"metadata.trashedAt": bson.M{"$exists": false},
		"metadata.deletedAt": bson.M{"$exists": false},
	})

	var eligible, hasThumb, noVideo, blocked int
	cursor, err := models.FileModel.FindRaw(ctx, bson.M{
		"status": models.FileStatusReady, "type": models.FileTypeVideo,
		"clonedFrom": bson.M{"$exists": false},
		"metadata.trashedAt": bson.M{"$exists": false},
		"metadata.deletedAt": bson.M{"$exists": false},
	}, options.Find().SetSort(bson.D{{Key: "createdAt", Value: 1}}).SetLimit(200))
	if err == nil {
		defer cursor.Close(ctx)
		for cursor.Next(ctx) {
			var file models.File
			if cursor.Decode(&file) != nil {
				continue
			}
			reason := spritesheetBlockReason(ctx, &file, excludeIDs)
			switch reason {
			case "":
				eligible++
				log.Printf("✅ [%s] eligible for spritesheet", file.Slug)
			case "has_thumbnail":
				hasThumb++
			case "pending_sprite_ingest":
				blocked++
				log.Printf("🔒 [%s] blocked: pending_sprite_ingest (await server-transfer)", file.Slug)
			case "no_video_media":
				noVideo++
			case "file_errors":
				blocked++
				log.Printf("🔒 [%s] blocked: file_errors", file.Slug)
			default:
				blocked++
				log.Printf("🔒 [%s] blocked: %s", file.Slug, reason)
			}
		}
	}

	log.Printf("💤 Idle — ready: %d | eligible: %d | has thumbnail: %d | no video: %d | blocked: %d",
		readyCount, eligible, hasThumb, noVideo, blocked)
	if blocked > 0 {
		log.Printf("💡 Clear stale lock: db.video_process.deleteOne({ fileId: \"...\", processType: \"spritesheet\" })")
	}
}

func findAndClaimFile(ctx context.Context) (*models.VideoProcess, *models.File, error) {
	excludeIDs := map[string]bool{}
	errorDocs, _ := models.FileErrorModel.Find(ctx, bson.M{"errorType": "spritesheet"})
	for _, e := range errorDocs {
		excludeIDs[e.FileID] = true
	}

	filter := bson.M{
		"status":             models.FileStatusReady,
		"type":               models.FileTypeVideo,
		"clonedFrom":         bson.M{"$exists": false},
		"metadata.trashedAt": bson.M{"$exists": false},
		"metadata.deletedAt": bson.M{"$exists": false},
	}
	if len(excludeIDs) > 0 {
		ids := make([]string, 0, len(excludeIDs))
		for id := range excludeIDs {
			ids = append(ids, id)
		}
		filter["_id"] = bson.M{"$nin": ids}
	}

	cursor, err := models.FileModel.FindRaw(ctx, filter, options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: 1}}).SetLimit(200))
	if err != nil {
		return nil, nil, err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var file models.File
		if err := cursor.Decode(&file); err != nil {
			continue
		}

		reason := spritesheetBlockReason(ctx, &file, excludeIDs)
		if reason == "own_processing_resume" || reason == "own_failed_retry" {
			continue // resumeOwnProcess handles these
		}
		if reason != "" {
			continue
		}

		clearStaleSpritesheetProcess(ctx, file.ID, file.Slug)
		if spritesheetBlockReason(ctx, &file, excludeIDs) != "" {
			continue
		}

		process, err := claimFile(ctx, &file)
		if err != nil {
			log.Printf("⚠️  [%s] Claim failed: %v", file.Slug, err)
			continue
		}
		return process, &file, nil
	}
	return nil, nil, nil
}

func claimFile(ctx context.Context, file *models.File) (*models.VideoProcess, error) {
	now := time.Now()
	processing := models.ProcessStatusProcessing
	pending := models.StepStatusPending
	process := &models.VideoProcess{
		ID: newUUID(), FileID: &file.ID, Slug: &file.Slug, WorkerID: &workerID,
		Status: &processing, SpaceID: file.SpaceID, ProcessType: models.ProcessTypeSpritesheet,
		Timeline: bson.M{
			"prepare":  bson.M{"status": pending},
			"generate": bson.M{"status": pending},
			"install":  bson.M{"status": pending},
		},
		CreatedAt: now, UpdatedAt: now,
	}
	if _, err := models.VideoProcessModel.Create(ctx, process); err != nil {
		return nil, err
	}
	log.Printf("🆕 [%s] Claimed for spritesheet (fileId=%s)", file.Slug, file.ID)
	return process, nil
}

func findSmallestVideo(ctx context.Context, fileID string) (*models.Media, error) {
	for _, res := range []string{models.Resolution360, models.Resolution480, models.Resolution720, models.Resolution1080, models.ResolutionOriginal} {
		media, err := models.MediaModel.FindOne(ctx, bson.M{
			"fileId": fileID, "type": models.MediaTypeVideo, "resolution": res,
			"deletedAt": bson.M{"$exists": false},
		})
		if err == nil {
			return media, nil
		}
	}
	return nil, fmt.Errorf("no video media for file %s", fileID)
}

func isColocatedStorage(storageID string) bool {
	return config.AppConfig.StorageId != "" &&
		config.AppConfig.StoragePath != "" &&
		storageID == config.AppConfig.StorageId
}

func resolveS3TempStorage(ctx context.Context) (*models.Storage, error) {
	return models.StorageModel.FindOne(ctx, bson.M{
		"enable": true, "status": models.StorageStatusOnline, "type": models.StorageTypeS3,
		"accepts": bson.M{"$all": []string{"temp", "video"}},
	}, options.FindOne().SetSort(bson.M{"capacity.percentage": 1}))
}

func runSpritesheet(ctx context.Context, process *models.VideoProcess) error {
	fileID := derefStr(process.FileID)
	slug := derefStr(process.Slug)

	procLogger := utils.NewProcessLogger(slug)
	defer procLogger.Close()

	exePath, _ := os.Executable()
	baseDir := filepath.Dir(exePath)
	if strings.Contains(exePath, "go-build") {
		baseDir, _ = os.Getwd()
	}
	workDir := filepath.Join(baseDir, "download", slug)
	os.MkdirAll(workDir, 0755)

	var success bool
	defer func() {
		if success {
			os.RemoveAll(workDir)
			log.Printf("🧹 [%s] Cleaned up temp dir", slug)
		}
	}()

	utils.LogMain("🖼️  [%s] START SPRITESHEET", slug)

	if hasPendingSpriteIngest(ctx, fileID) {
		log.Printf("⏭️  [%s] sprite.zip ingest pending — skip (await server-transfer)", slug)
		success = true
		models.VideoProcessModel.DeleteByID(ctx, process.ID)
		return nil
	}
	if repairOrphanSpriteOnS3(ctx, fileID, slug) {
		success = true
		models.VideoProcessModel.DeleteByID(ctx, process.ID)
		return nil
	}
	if hasThumbnailMedia(ctx, fileID) {
		log.Printf("⏭️  [%s] thumbnail media exists — skip", slug)
		success = true
		models.VideoProcessModel.DeleteByID(ctx, process.ID)
		return nil
	}

	// ─── PREPARE: resolve video input ───────────────────────────
	startStep(ctx, process.ID, "prepare")

	videoMedia, err := findSmallestVideo(ctx, fileID)
	if err != nil {
		failProcess(ctx, process.ID, slug, err.Error())
		return err
	}

	sourceStorageID := derefStr(videoMedia.StorageID)
	sourceStorage, err := models.StorageModel.FindByID(ctx, sourceStorageID)
	if err != nil {
		failProcess(ctx, process.ID, slug, "source storage not found")
		return err
	}

	videoFileName := derefStr(videoMedia.FileName)
	inputPath, downloaded, err := resolveVideoInput(ctx, process, slug, fileID, workDir, videoMedia, sourceStorage, videoFileName)
	if err != nil {
		failProcess(ctx, process.ID, slug, fmt.Sprintf("prepare input: %v", err))
		return err
	}

	duration := fileDurationFromProcess(ctx, fileID)
	if duration <= 0 {
		if info, probeErr := spritesheet.ProbeVideoInfo(inputPath); probeErr == nil {
			duration = info.DurationF
		}
	}

	completeStep(ctx, process.ID, "prepare")
	updateOverallPercent(ctx, process.ID, 15)
	log.Printf("✅ [%s] Input ready: %s (%.1fs)", slug, filepath.Base(inputPath), duration)

	// ─── GENERATE sprite sheets ─────────────────────────────────
	startStep(ctx, process.ID, "generate")
	updateOverallPercent(ctx, process.ID, 20)

	result, err := spritesheet.Generate(inputPath, workDir, duration)
	if err != nil {
		failProcess(ctx, process.ID, slug, fmt.Sprintf("generate: %v", err))
		return err
	}
	os.Remove(filepath.Join(result.SpriteDir, "cropped_last.jpg"))

	if downloaded {
		os.Remove(inputPath)
	}

	completeStep(ctx, process.ID, "generate")
	updateOverallPercent(ctx, process.ID, 70)
	log.Printf("✅ [%s] Generated %d sprite files", slug, len(result.SpriteFiles))

	var totalSpriteSize int64
	for _, name := range result.SpriteFiles {
		totalSpriteSize += spritesheet.GetFileSize(filepath.Join(result.SpriteDir, name))
	}

	// ─── INSTALL: local storage or S3 tmp ───────────────────────
	startStep(ctx, process.ID, "install")
	updateOverallPercent(ctx, process.ID, 75)

	if isColocatedStorage(sourceStorageID) {
		if err := installSpriteLocal(fileID, result.SpriteDir); err != nil {
			failProcess(ctx, process.ID, slug, fmt.Sprintf("install local: %v", err))
			return err
		}
		if err := createThumbnailMedia(ctx, fileID, slug, sourceStorageID, totalSpriteSize); err != nil {
			failProcess(ctx, process.ID, slug, fmt.Sprintf("create media: %v", err))
			return err
		}
		log.Printf("✅ [%s] Installed sprite/ on local storage", slug)
	} else {
		zipPath := filepath.Join(workDir, models.SpriteZipName)
		if err := archive.ZipDir(result.SpriteDir, zipPath); err != nil {
			failProcess(ctx, process.ID, slug, fmt.Sprintf("zip: %v", err))
			return err
		}
		s3Storage, err := resolveS3TempStorage(ctx)
		if err != nil {
			failProcess(ctx, process.ID, slug, fmt.Sprintf("no S3 temp storage: %v", err))
			return err
		}
		objectKey := fileID + "/" + models.SpriteZipName
		zipInfo, _ := os.Stat(zipPath)
		var zipSize int64
		if zipInfo != nil {
			zipSize = zipInfo.Size()
		}
		if err := uploader.UploadToS3(s3Storage, zipPath, objectKey, "application/zip", func(done, total int64) {
			if total > 0 {
				pct := 75.0 + float64(done)/float64(total)*20.0
				updateOverallPercent(ctx, process.ID, pct)
			}
		}); err != nil {
			failProcess(ctx, process.ID, slug, fmt.Sprintf("S3 upload: %v", err))
			return err
		}
		if err := createSpriteIngest(ctx, fileID, s3Storage, zipSize); err != nil {
			failProcess(ctx, process.ID, slug, fmt.Sprintf("create ingest: %v", err))
			return err
		}
		log.Printf("✅ [%s] Uploaded %s + ingest (server-transfer will install)", slug, objectKey)
	}

	completeStep(ctx, process.ID, "install")
	updateOverallPercent(ctx, process.ID, 100)

	success = true
	models.VideoProcessModel.DeleteByID(ctx, process.ID)
	utils.LogMain("✅ [%s] SPRITESHEET COMPLETE", slug)
	return nil
}

func resolveVideoInput(
	ctx context.Context,
	process *models.VideoProcess,
	slug, fileID, workDir string,
	videoMedia *models.Media,
	sourceStorage *models.Storage,
	videoFileName string,
) (inputPath string, downloaded bool, err error) {
	sourceStorageID := derefStr(videoMedia.StorageID)

	if isColocatedStorage(sourceStorageID) {
		localPath := filepath.Join(config.AppConfig.StoragePath, fileID, videoFileName)
		if _, statErr := os.Stat(localPath); statErr == nil {
			log.Printf("📂 [%s] Using local storage file: %s", slug, localPath)
			return localPath, false, nil
		}
		log.Printf("⚠️  [%s] Local file missing, falling back to HTTP download", slug)
	}

	hostPort := sourceStorage.GetHostPort()
	if hostPort == "" {
		return "", false, fmt.Errorf("storage has no host")
	}

	destPath := filepath.Join(workDir, videoFileName)
	url := fmt.Sprintf("http://%s/%s.mp4", hostPort, videoMedia.Slug)
	log.Printf("📥 [%s] Downloading %s", slug, url)

	err = downloader.DownloadURL(ctx, url, destPath, func(done, total int64) {
		if total > 0 {
			pct := float64(done) / float64(total) * 15.0
			updateTimelineStep(ctx, process.ID, "prepare", models.StepStatusProcessing, pct)
			updateOverallPercent(ctx, process.ID, pct)
		}
	})
	if err != nil {
		return "", false, err
	}
	return destPath, true, nil
}

func installSpriteLocal(fileID, spriteDir string) error {
	destDir := filepath.Join(config.AppConfig.StoragePath, fileID, "sprite")
	os.MkdirAll(destDir, 0755)

	entries, err := os.ReadDir(spriteDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		src := filepath.Join(spriteDir, entry.Name())
		_, err := uploader.MoveFileLocal(config.AppConfig.StoragePath, fileID, src, "sprite/"+entry.Name(), nil)
		if err != nil {
			return fmt.Errorf("move %s: %w", entry.Name(), err)
		}
	}
	return nil
}

func createThumbnailMedia(ctx context.Context, fileID, slug, storageID string, totalSize int64) error {
	count, _ := models.MediaModel.CountDocuments(ctx, bson.M{
		"fileId": fileID, "type": models.MediaTypeThumbnail,
		"deletedAt": bson.M{"$exists": false},
	})
	if count > 0 {
		return nil
	}

	now := time.Now()
	thumbFn := models.SpriteVTTName
	storageIDPtr := storageID
	media := models.Media{
		ID: newUUID(), Type: models.MediaTypeThumbnail, FileName: &thumbFn,
		StorageID: &storageIDPtr, Slug: utils.RandomString(11, false), FileID: &fileID,
		Metadata: &models.MediaMetadata{Size: totalSize}, CreatedAt: now, UpdatedAt: now,
	}
	models.MediaModel.Create(ctx, &media)
	cloneMediaToClonedFiles(ctx, fileID, media, slug)
	log.Printf("✅ [%s] Created thumbnail media record", slug)
	return nil
}

func fileDurationFromProcess(ctx context.Context, fileID string) float64 {
	file, err := models.FileModel.FindByID(ctx, fileID)
	if err != nil || file.Metadata == nil || file.Metadata.Duration == nil {
		return 0
	}
	return float64(*file.Metadata.Duration)
}
