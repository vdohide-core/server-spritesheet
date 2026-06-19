package spritesheet

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// VideoInfo holds basic video metadata from ffprobe.
type VideoInfo struct {
	Width     int64
	Height    int64
	DurationF float64
}

// ProbeVideoInfo reads duration and dimensions from a video file.
func ProbeVideoInfo(filePath string) (*VideoInfo, error) {
	info := &VideoInfo{}

	cmd := exec.Command("ffprobe",
		"-v", "error", "-select_streams", "v:0",
		"-show_entries", "stream=width,height",
		"-of", "csv=s=x:p=0", filePath)
	if output, err := cmd.Output(); err == nil {
		parts := strings.Split(strings.TrimSpace(string(output)), "x")
		if len(parts) == 2 {
			info.Width, _ = strconv.ParseInt(parts[0], 10, 64)
			info.Height, _ = strconv.ParseInt(parts[1], 10, 64)
		}
	}

	cmd = exec.Command("ffprobe",
		"-v", "error", "-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1", filePath)
	if output, err := cmd.Output(); err == nil {
		info.DurationF, _ = strconv.ParseFloat(strings.TrimSpace(string(output)), 64)
	}

	if info.DurationF <= 0 && info.Width == 0 {
		return nil, fmt.Errorf("ffprobe failed for %s", filePath)
	}
	return info, nil
}
