package database

import (
	"context"
	"log"
	"time"

	"server-spritesheet/internal/config"
	"server-spritesheet/internal/db/models"

	"github.com/zergolf1994/goose"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Connect establishes a connection to MongoDB via goose ODM.
func Connect() error {
	uri := config.AppConfig.MongoURI
	if err := goose.Connect(uri); err != nil {
		return err
	}
	EnsureIndexes()
	return nil
}

// Disconnect closes the MongoDB connection.
func Disconnect() {
	if goose.Client() != nil {
		if err := goose.Close(); err != nil {
			log.Printf("⚠️ Error disconnecting from MongoDB: %v", err)
		} else {
			log.Println("🔌 Disconnected from MongoDB")
		}
	}
}

// ─── Indexes ──────────────────────────────────────────────────

// EnsureIndexes creates required indexes for concurrency safety.
func EnsureIndexes() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	vpCol := models.VideoProcessModel.Col()

	// Drop old single-field indexes (one-time migration)
	vpCol.Indexes().DropOne(ctx, "postId_1")
	vpCol.Indexes().DropOne(ctx, "fileId_1")

	// Compound unique index: one process per file per processType
	// This allows multiple process types (download, subtitle, etc.) for the same file
	_, err := vpCol.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "fileId", Value: 1}, {Key: "processType", Value: 1}},
		Options: options.Index().SetUnique(true).SetSparse(true),
	})
	if err != nil {
		log.Printf("⚠️  Index creation warning: %v", err)
	} else {
		log.Printf("✅ Unique index on video_process.{fileId,processType} ensured")
	}
}
