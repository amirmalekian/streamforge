package downloader

import (
	"context"
	"os/exec"
	"testing"
	"time"
)

func TestYTDLPDownloader_Download_Integration(t *testing.T) {
	if !isYTDLPAvailable() {
		t.Skip("yt-dlp not available, skipping integration test")
	}

	tempDir := t.TempDir()
	downloader := NewYTDLPDownloader(tempDir)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	item := Item{
		ID:  "test_video",
		URL: "https://www.youtube.com/watch?v=jNQXAC9IVRw",
	}

	result, err := downloader.DownloadWithContext(ctx, item)
	if err != nil {
		t.Fatalf("Download failed: %v", err)
	}

	if result.ID == "" {
		t.Error("Expected non-empty ID in result")
	}

	if result.FilePath == "" {
		t.Error("Expected non-empty FilePath in result")
	}

	if result.Size <= 0 {
		t.Error("Expected positive file size")
	}
}

func TestYTDLPDownloader_Download_Cancelled_Integration(t *testing.T) {
	if !isYTDLPAvailable() {
		t.Skip("yt-dlp not available, skipping integration test")
	}

	tempDir := t.TempDir()
	downloader := NewYTDLPDownloader(tempDir)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	item := Item{
		ID:  "test_cancel",
		URL: "https://www.youtube.com/watch?v=jNQXAC9IVRw",
	}

	_, err := downloader.DownloadWithContext(ctx, item)
	if err == nil {
		t.Error("Expected error when context is cancelled")
	}
}

func TestYTDLPDownloader_GetMetadata_Integration(t *testing.T) {
	if !isYTDLPAvailable() {
		t.Skip("yt-dlp not available, skipping integration test")
	}

	tempDir := t.TempDir()
	downloader := NewYTDLPDownloader(tempDir)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	metadata, err := downloader.GetMetadata(ctx, "https://www.youtube.com/watch?v=jNQXAC9IVRw")
	if err != nil {
		t.Fatalf("GetMetadata failed: %v", err)
	}

	if metadata.ID == "" {
		t.Error("Expected non-empty ID in metadata")
	}

	if metadata.Title == "" {
		t.Error("Expected non-empty Title in metadata")
	}

	if metadata.Duration <= 0 {
		t.Error("Expected positive Duration in metadata")
	}
}

func TestYTDLPDownloader_Download_InvalidURL_Integration(t *testing.T) {
	if !isYTDLPAvailable() {
		t.Skip("yt-dlp not available, skipping integration test")
	}

	tempDir := t.TempDir()
	downloader := NewYTDLPDownloader(tempDir)

	ctx := context.Background()
	item := Item{
		ID:  "test_invalid",
		URL: "not-a-valid-url",
	}

	_, err := downloader.DownloadWithContext(ctx, item)
	if err == nil {
		t.Error("Expected error for invalid URL")
	}
}

func isYTDLPAvailable() bool {
	_, err := exec.LookPath("yt-dlp")
	return err == nil
}
