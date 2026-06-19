package spritesheet

import (
	"fmt"
	"log"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	spriteCols     = 6
	spriteMaxRows  = 6
	spriteInterval = 1.0 // 1000ms per frame
)

// Result contains generated sprite output.
type Result struct {
	SpriteDir   string
	SpriteFiles []string
	VTTFile     string
}

// Generate creates sprite sheets (6x6, 1s interval) under {outputDir}/sprite/.
func Generate(inputPath, outputDir string, duration float64) (*Result, error) {
	if duration <= 0 {
		info, err := ProbeVideoInfo(inputPath)
		if err != nil || info.DurationF <= 0 {
			return nil, fmt.Errorf("invalid duration: %.2f", duration)
		}
		duration = info.DurationF
	}

	spriteDir := filepath.Join(outputDir, "sprite")
	os.MkdirAll(spriteDir, 0755)

	framesPerSheet := spriteCols * spriteMaxRows
	totalFrames := int(math.Floor(duration / spriteInterval))
	if totalFrames < 1 {
		totalFrames = 1
	}

	thumbWidth, thumbHeight := calcThumbSize(inputPath)
	log.Printf("🧠 Generating %d thumbnails (interval = %.0fms, size = %dx%d)", totalFrames, spriteInterval*1000, thumbWidth, thumbHeight)

	fpsFilter := fmt.Sprintf("fps=1/%.2f", spriteInterval)
	scaleFilter := fmt.Sprintf("scale=%d:%d", thumbWidth, thumbHeight)
	tileFilter := fmt.Sprintf("tile=%dx%d", spriteCols, spriteMaxRows)
	spritePattern := filepath.Join(spriteDir, "sprite-%d.jpg")

	cmd := exec.Command("ffmpeg",
		"-y",
		"-i", inputPath,
		"-vf", fmt.Sprintf("%s,%s,%s", fpsFilter, scaleFilter, tileFilter),
		"-q:v", "9",
		spritePattern,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("sprite generation failed: %w\n%s", err, string(output))
	}

	var spriteFiles []string
	for i := 1; i <= 1000; i++ {
		name := fmt.Sprintf("sprite-%d.jpg", i)
		if _, err := os.Stat(filepath.Join(spriteDir, name)); err != nil {
			break
		}
		spriteFiles = append(spriteFiles, name)
	}
	if len(spriteFiles) == 0 {
		return nil, fmt.Errorf("no sprite files generated")
	}

	firstPath := filepath.Join(spriteDir, spriteFiles[0])
	if actualW, actualH := probeImageDimensions(firstPath); actualW > 0 && actualH > 0 {
		realW := actualW / spriteCols
		realH := actualH / spriteMaxRows
		if realW != thumbWidth || realH != thumbHeight {
			thumbWidth, thumbHeight = realW, realH
		}
	}

	sheetsCount := len(spriteFiles)
	framesInLastSheet := totalFrames - (sheetsCount-1)*framesPerSheet
	if framesInLastSheet <= 0 {
		framesInLastSheet = 1
	}
	lastSheetRows := int(math.Ceil(float64(framesInLastSheet) / float64(spriteCols)))
	if lastSheetRows < spriteMaxRows && lastSheetRows > 0 {
		lastName := spriteFiles[sheetsCount-1]
		lastPath := filepath.Join(spriteDir, lastName)
		croppedPath := filepath.Join(spriteDir, "cropped_last.jpg")
		cropW := thumbWidth * spriteCols
		cropH := thumbHeight * lastSheetRows

		cropCmd := exec.Command("ffmpeg",
			"-y", "-i", lastPath,
			"-vf", fmt.Sprintf("crop=%d:%d:0:0", cropW, cropH),
			"-q:v", "5", croppedPath,
		)
		if cropOutput, cropErr := cropCmd.CombinedOutput(); cropErr != nil {
			log.Printf("⚠️ Failed to crop last sprite sheet: %s", string(cropOutput))
		} else {
			os.Remove(lastPath)
			os.Rename(croppedPath, lastPath)
		}
	}

	vttContent := generateVTT(spriteFiles, spriteInterval, thumbWidth, thumbHeight, totalFrames, spriteCols, framesPerSheet)
	vttPath := filepath.Join(spriteDir, "sprite.vtt")
	if err := os.WriteFile(vttPath, []byte(vttContent), 0644); err != nil {
		return nil, fmt.Errorf("write sprite.vtt: %w", err)
	}
	spriteFiles = append(spriteFiles, "sprite.vtt")

	return &Result{
		SpriteDir:   spriteDir,
		SpriteFiles: spriteFiles,
		VTTFile:     "sprite.vtt",
	}, nil
}

func calcThumbSize(inputPath string) (int, int) {
	const thumbHeight = 168
	info, err := ProbeVideoInfo(inputPath)
	if err != nil || info.Width <= 0 || info.Height <= 0 {
		w := thumbHeight * 16 / 9 // default 16:9
		if w%2 != 0 {
			w++
		}
		return w, thumbHeight
	}
	w := int(float64(info.Width) * float64(thumbHeight) / float64(info.Height))
	if w%2 != 0 {
		w++
	}
	return w, thumbHeight
}

func generateVTT(spriteFiles []string, interval float64, thumbW, thumbH, totalFrames, cols, framesPerSheet int) string {
	var sb strings.Builder
	sb.WriteString("WEBVTT\n\n")
	frameIndex := 0
	for _, fileName := range spriteFiles {
		remaining := totalFrames - frameIndex
		framesInSheet := framesPerSheet
		if remaining < framesPerSheet {
			framesInSheet = remaining
		}
		if framesInSheet <= 0 {
			break
		}
		for i := 0; i < framesInSheet; i++ {
			col := i % cols
			row := i / cols
			start := float64(frameIndex) * interval
			end := float64(frameIndex+1) * interval
			sb.WriteString(fmt.Sprintf("%s --> %s\n", formatVTTTime(start), formatVTTTime(end)))
			sb.WriteString(fmt.Sprintf("%s#xywh=%d,%d,%d,%d\n\n",
				fileName, col*thumbW, row*thumbH, thumbW, thumbH))
			frameIndex++
		}
	}
	return sb.String()
}

func formatVTTTime(seconds float64) string {
	if seconds < 0 {
		seconds = 0
	}
	h := int(seconds) / 3600
	m := (int(seconds) % 3600) / 60
	s := int(seconds) % 60
	ms := int((seconds - float64(int(seconds))) * 1000)
	return fmt.Sprintf("%02d:%02d:%02d.%03d", h, m, s, ms)
}

func probeImageDimensions(imagePath string) (int, int) {
	cmd := exec.Command("ffprobe",
		"-v", "error", "-select_streams", "v:0",
		"-show_entries", "stream=width,height", "-of", "csv=s=x:p=0", imagePath)
	output, err := cmd.Output()
	if err != nil {
		return 0, 0
	}
	parts := strings.Split(strings.TrimSpace(string(output)), "x")
	if len(parts) != 2 {
		return 0, 0
	}
	w, _ := strconv.Atoi(parts[0])
	h, _ := strconv.Atoi(parts[1])
	return w, h
}

// GetFileSize returns file size in bytes.
func GetFileSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}

// CheckFFmpeg verifies ffmpeg is available.
func CheckFFmpeg() error {
	cmd := exec.Command("ffmpeg", "-version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg not found in PATH")
	}
	return nil
}
