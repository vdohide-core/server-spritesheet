package downloader

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// DownloadURL saves an HTTP resource to destPath.
func DownloadURL(ctx context.Context, url, destPath string, onProgress func(done, total int64)) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	client := &http.Client{Timeout: 2 * time.Hour}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}

	tmpPath := destPath + ".part"
	out, err := os.Create(tmpPath)
	if err != nil {
		return err
	}

	total := resp.ContentLength
	var done int64
	buf := make([]byte, 1024*1024)
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, wErr := out.Write(buf[:n]); wErr != nil {
				out.Close()
				os.Remove(tmpPath)
				return wErr
			}
			done += int64(n)
			if onProgress != nil {
				onProgress(done, total)
			}
		}
		if readErr != nil {
			if readErr == io.EOF {
				break
			}
			out.Close()
			os.Remove(tmpPath)
			return readErr
		}
	}

	if err := out.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}
	return os.Rename(tmpPath, destPath)
}
