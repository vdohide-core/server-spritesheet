package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"server-spritesheet/internal/config"
	"server-spritesheet/internal/db/database"
	"server-spritesheet/internal/db/models"
	"server-spritesheet/internal/handlers"
	"server-spritesheet/internal/logger"
	"server-spritesheet/internal/middleware"
	"server-spritesheet/internal/spritesheet"
	"server-spritesheet/internal/utils"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

var workerID string

func main() {
	config.Load()
	workerID = utils.GenerateWorkerID()
	log.Printf("Starting Server Spritesheet [Worker: %s]", workerID)

	logCloser, err := logger.Init(config.AppConfig.LogPath)
	if err != nil {
		log.Printf("⚠️ File logging disabled: %v", err)
	} else {
		defer logCloser.Close()
		log.Printf("📝 Logging to: %s", config.AppConfig.LogPath)
	}

	if err := database.Connect(); err != nil {
		log.Printf("ERROR: MongoDB: %v", err)
		time.Sleep(5 * time.Second)
		os.Exit(1)
	}
	defer database.Disconnect()
	log.Println("✅ MongoDB connected")

	if err := spritesheet.CheckFFmpeg(); err != nil {
		log.Printf("⚠️ ffmpeg not found: %v", err)
	}

	port := config.AppConfig.Port
	if port == "" {
		port = "8084"
	}

	logDir := filepath.Dir(config.AppConfig.LogPath)
	h := handlers.NewHandler(handlers.Handler{LogDir: logDir})
	go handlers.GlobalHub.Run()
	go handlers.WatchLogDir(logDir)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok","service":"server-spritesheet","worker":"%s"}`, workerID)
	})
	mux.HandleFunc("/logs", h.HandleLogList)
	mux.HandleFunc("/logs/", h.HandleLogFile)
	mux.HandleFunc("/ui", h.HandleUI)
	mux.HandleFunc("/ws", h.HandleWS)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})

	go func() {
		ln, err := net.Listen("tcp", ":"+port)
		if err != nil {
			log.Printf("📋 Log viewer skipped (port %s in use)", port)
			return
		}
		server := &http.Server{Handler: middleware.CORS(mux)}
		log.Printf("🌐 Log viewer: http://localhost:%s/ui", port)
		if err := server.Serve(ln); err != http.ErrServerClosed {
			log.Printf("⚠️ HTTP server error: %v", err)
		}
	}()

	go startHeartbeat(workerID)
	startWorkerLoop()
}

func startWorkerLoop() {
	log.Println("⚡ Worker Mode: Polling for spritesheet jobs...")
	log.Printf("🆔 Worker ID: %s", workerID)
	if config.AppConfig.StorageId != "" {
		log.Printf("📦 Co-located storage: %s → %s", config.AppConfig.StorageId, config.AppConfig.StoragePath)
	}

	utils.CleanOldLogs()

	ctx := context.Background()
	if isSpritesheetEnabled(ctx) {
		log.Println("✅ spritesheet_enabled=true")
	} else {
		log.Println("⏸️  spritesheet_enabled=false — enable in db.settings")
	}
	const pollBusy = 5 * time.Second
	const pollIdle = 30 * time.Second
	idleRounds := 0
	var lastDisabledLog time.Time

	for {
		if !isSpritesheetEnabled(ctx) {
			if time.Since(lastDisabledLog) > 5*time.Minute {
				log.Println("⏸️  spritesheet_enabled=false — set db.settings.spritesheet_enabled=true")
				lastDisabledLog = time.Now()
			}
			time.Sleep(pollIdle)
			continue
		}
		if !isWorkerEnabled(ctx) {
			if time.Since(lastDisabledLog) > 5*time.Minute {
				log.Printf("⏸️  worker %s disabled in db.workers", workerID)
				lastDisabledLog = time.Now()
			}
			time.Sleep(pollIdle)
			continue
		}
		if processNextJob(ctx) {
			idleRounds = 0
			time.Sleep(pollBusy)
		} else {
			idleRounds++
			if idleRounds == 1 || idleRounds%10 == 0 {
				logIdleDiagnostics(ctx)
			}
			time.Sleep(pollIdle)
		}
	}
}

func isSpritesheetEnabled(ctx context.Context) bool {
	setting, err := models.SettingModel.FindOne(ctx, bson.M{"name": models.SettingSpritesheetEnabled})
	if err != nil {
		if err != mongo.ErrNoDocuments {
			log.Printf("⚠️  settings lookup failed: %v", err)
		}
		return false
	}
	return setting.GetBool(false)
}

func isWorkerEnabled(ctx context.Context) bool {
	worker, err := models.WorkerModel.FindOne(ctx, bson.M{"workerId": workerID})
	if err != nil {
		return true
	}
	return worker.Enable
}

func processNextJob(ctx context.Context) bool {
	cleanupMaxRetryProcesses(ctx)

	if process := resumeOwnProcess(ctx); process != nil {
		slug := derefStr(process.Slug)
		if err := runSpritesheet(ctx, process); err != nil {
			log.Printf("❌ Resume failed: %s - %v", slug, err)
		}
		return true
	}

	process, file, err := findAndClaimFile(ctx)
	if err == nil && process != nil {
		slug := derefStr(process.Slug)
		log.Printf("🖼️  New spritesheet job: [%s] %s", slug, file.Name)
		if err := runSpritesheet(ctx, process); err != nil {
			log.Printf("❌ Failed: %s - %v", slug, err)
		}
		return true
	}
	return false
}

func resumeOwnProcess(ctx context.Context) *models.VideoProcess {
	process, err := models.VideoProcessModel.FindOne(ctx, bson.M{
		"workerId": workerID, "status": models.ProcessStatusProcessing,
		"processType": models.ProcessTypeSpritesheet,
	})
	if err == nil {
		log.Printf("🔄 [%s] Resuming interrupted process", derefStr(process.Slug))
		return process
	}

	failed, err := models.VideoProcessModel.FindOne(ctx, bson.M{
		"workerId": workerID, "status": models.ProcessStatusFailed,
		"processType": models.ProcessTypeSpritesheet, "retryCount": bson.M{"$lt": 3},
	})
	if err == nil {
		slug := derefStr(failed.Slug)
		retryNum := 0
		if failed.RetryCount != nil {
			retryNum = *failed.RetryCount
		}
		waitSec := 30
		if retryNum >= 2 {
			waitSec = 60
		}
		log.Printf("🔁 [%s] Retrying (attempt %d/3) — waiting %ds...", slug, retryNum+1, waitSec)
		time.Sleep(time.Duration(waitSec) * time.Second)
		models.VideoProcessModel.Col().UpdateOne(ctx, bson.M{"_id": failed.ID}, bson.M{"$set": bson.M{
			"status": models.ProcessStatusProcessing, "error": "", "updatedAt": time.Now(),
		}})
		status := models.ProcessStatusProcessing
		failed.Status = &status
		return failed
	}
	return nil
}

func cleanupMaxRetryProcesses(ctx context.Context) {
	processes, _ := models.VideoProcessModel.Find(ctx, bson.M{
		"workerId": workerID, "status": models.ProcessStatusFailed,
		"processType": models.ProcessTypeSpritesheet, "retryCount": bson.M{"$gte": 3},
	})
	for _, pf := range processes {
		slug := derefStr(pf.Slug)
		fileID := derefStr(pf.FileID)
		removeDownloadDir(slug)
		models.VideoProcessModel.DeleteByID(ctx, pf.ID)
		errMsg := ""
		if pf.Error != nil {
			errMsg = *pf.Error
		}
		existing, _ := models.FileErrorModel.CountDocuments(ctx, bson.M{
			"fileId": fileID, "errorType": "spritesheet",
		})
		if existing == 0 {
			models.FileErrorModel.Create(ctx, &models.FileError{
				FileID: fileID, ErrorType: "spritesheet", Error: errMsg, Slug: slug, WorkerID: workerID,
			})
		}
		log.Printf("🗑️  [%s] Max retries — marked permanently_failed", slug)
	}
}

func removeDownloadDir(slug string) {
	exePath, _ := os.Executable()
	baseDir := filepath.Dir(exePath)
	if strings.Contains(exePath, "go-build") {
		baseDir, _ = os.Getwd()
	}
	os.RemoveAll(filepath.Join(baseDir, "download", slug))
}
