package downloader

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestYTDLPDownloader_OutputDirectory(t *testing.T) {
	customDir := "/custom/output/path"
	downloader := NewYTDLPDownloader(customDir)

	if downloader.outputDir != customDir {
		t.Errorf("Expected outputDir to be %s, got %s", customDir, downloader.outputDir)
	}
}

func TestYTDLPDownloader_DefaultOutputDirectory(t *testing.T) {
	downloader := NewYTDLPDownloader("")

	if downloader.outputDir != "/tmp" {
		t.Errorf("Expected default outputDir to be /tmp, got %s", downloader.outputDir)
	}
}

func TestYTDLPDownloader_SetTimeout(t *testing.T) {
	downloader := NewYTDLPDownloader("/tmp")
	customTimeout := 10 * time.Minute

	downloader.SetTimeout(customTimeout)

	if downloader.timeout != customTimeout {
		t.Errorf("Expected timeout to be %v, got %v", customTimeout, downloader.timeout)
	}
}

func TestYTDLPDownloader_DefaultTimeout(t *testing.T) {
	downloader := NewYTDLPDownloader("/tmp")

	expected := 30 * time.Minute
	if downloader.timeout != expected {
		t.Errorf("Expected default timeout %v, got %v", expected, downloader.timeout)
	}
}

func TestYTDLPDownloader_parseResult(t *testing.T) {
	tempDir := t.TempDir()
	downloader := NewYTDLPDownloader(tempDir)

	infoJSON := `{
		"id": "test123",
		"title": "Test Video",
		"duration": 120,
		"format": "mp4",
		"ext": "mp4",
		"filesize": 1024000
	}`
	infoPath := filepath.Join(tempDir, "test123.info.json")
	if err := os.WriteFile(infoPath, []byte(infoJSON), 0644); err != nil {
		t.Fatalf("Failed to write test info.json: %v", err)
	}

	videoPath := filepath.Join(tempDir, "test123.mp4")
	if err := os.WriteFile(videoPath, []byte("fake video content"), 0644); err != nil {
		t.Fatalf("Failed to write test video file: %v", err)
	}

	result, err := downloader.parseResult("test123")
	if err != nil {
		t.Fatalf("parseResult failed: %v", err)
	}

	if result.ID != "test123" {
		t.Errorf("Expected ID test123, got %s", result.ID)
	}

	if result.FilePath != videoPath {
		t.Errorf("Expected FilePath %s, got %s", videoPath, result.FilePath)
	}

	if result.Size != 18 {
		t.Errorf("Expected Size 18, got %d", result.Size)
	}
}

func TestYTDLPDownloader_parseResult_NotFound(t *testing.T) {
	tempDir := t.TempDir()
	downloader := NewYTDLPDownloader(tempDir)

	_, err := downloader.parseResult("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}

func TestYTDLPDownloader_parseResult_WrongID(t *testing.T) {
	tempDir := t.TempDir()
	downloader := NewYTDLPDownloader(tempDir)

	infoJSON := `{"id": "other123", "title": "Other Video", "duration": 100, "format": "mp4", "ext": "mp4", "filesize": 5000}`
	infoPath := filepath.Join(tempDir, "other123.info.json")
	if err := os.WriteFile(infoPath, []byte(infoJSON), 0644); err != nil {
		t.Fatalf("Failed to write test info.json: %v", err)
	}

	videoPath := filepath.Join(tempDir, "other123.mp4")
	if err := os.WriteFile(videoPath, []byte("video content"), 0644); err != nil {
		t.Fatalf("Failed to write test video file: %v", err)
	}

	_, err := downloader.parseResult("test123")
	if err == nil {
		t.Error("Expected error when ID doesn't match")
	}
}

func TestYTDLPDownloader_parseResult_InvalidJSON(t *testing.T) {
	tempDir := t.TempDir()
	downloader := NewYTDLPDownloader(tempDir)

	infoPath := filepath.Join(tempDir, "test123.info.json")
	if err := os.WriteFile(infoPath, []byte("invalid json"), 0644); err != nil {
		t.Fatalf("Failed to write test info.json: %v", err)
	}

	videoPath := filepath.Join(tempDir, "test123.mp4")
	if err := os.WriteFile(videoPath, []byte("video content"), 0644); err != nil {
		t.Fatalf("Failed to write test video file: %v", err)
	}

	_, err := downloader.parseResult("test123")
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestYTDLPDownloader_parseResult_NoVideoFile(t *testing.T) {
	tempDir := t.TempDir()
	downloader := NewYTDLPDownloader(tempDir)

	infoJSON := `{"id": "test123", "title": "Test Video", "duration": 120, "format": "mp4", "ext": "mp4", "filesize": 1024000}`
	infoPath := filepath.Join(tempDir, "test123.info.json")
	if err := os.WriteFile(infoPath, []byte(infoJSON), 0644); err != nil {
		t.Fatalf("Failed to write test info.json: %v", err)
	}

	_, err := downloader.parseResult("test123")
	if err == nil {
		t.Error("Expected error when video file doesn't exist")
	}
}

type fakeCommandRunner struct {
	runFunc            func(ctx context.Context, name string, args ...string) ([]byte, error)
	combinedOutputFunc func(ctx context.Context, name string, args ...string) ([]byte, error)
}

func (f *fakeCommandRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	if f.runFunc != nil {
		return f.runFunc(ctx, name, args...)
	}
	return nil, nil
}

func (f *fakeCommandRunner) CombinedOutput(ctx context.Context, name string, args ...string) ([]byte, error) {
	if f.combinedOutputFunc != nil {
		return f.combinedOutputFunc(ctx, name, args...)
	}
	return nil, nil
}

func TestYTDLPDownloader_GetMetadata_Success(t *testing.T) {
	tempDir := t.TempDir()
	downloader := NewYTDLPDownloader(tempDir)

	downloader.setCommandRunner(&fakeCommandRunner{
		runFunc: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			if name != "yt-dlp" {
				return nil, errors.New("unexpected command")
			}
			return []byte(`{"id": "dQw4w9WgXcQ", "title": "Rick Astley - Never Gonna Give You Up", "duration": 212, "format": "mp4", "ext": "mp4", "filesize": 5000000}`), nil
		},
	})

	ctx := context.Background()
	metadata, err := downloader.GetMetadata(ctx, "https://youtube.com/watch?v=dQw4w9WgXcQ")
	if err != nil {
		t.Fatalf("GetMetadata failed: %v", err)
	}

	if metadata.ID != "dQw4w9WgXcQ" {
		t.Errorf("Expected ID dQw4w9WgXcQ, got %s", metadata.ID)
	}
	if metadata.Title != "Rick Astley - Never Gonna Give You Up" {
		t.Errorf("Unexpected title: %s", metadata.Title)
	}
	if metadata.Duration != 212 {
		t.Errorf("Expected duration 212, got %d", metadata.Duration)
	}
	if metadata.Size != 5000000 {
		t.Errorf("Expected size 5000000, got %d", metadata.Size)
	}
}

func TestYTDLPDownloader_GetMetadata_ContextCancelled(t *testing.T) {
	tempDir := t.TempDir()
	downloader := NewYTDLPDownloader(tempDir)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	downloader.setCommandRunner(&fakeCommandRunner{
		runFunc: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return nil, context.Canceled
		},
	})

	_, err := downloader.GetMetadata(ctx, "https://youtube.com/watch?v=test")
	if err == nil {
		t.Error("Expected error when context is cancelled")
	}
	if err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Expected context cancellation error, got: %v", err)
	}
}

func TestYTDLPDownloader_GetMetadata_ExecutionError(t *testing.T) {
	tempDir := t.TempDir()
	downloader := NewYTDLPDownloader(tempDir)

	downloader.setCommandRunner(&fakeCommandRunner{
		runFunc: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return nil, errors.New("yt-dlp not found")
		},
	})

	ctx := context.Background()
	_, err := downloader.GetMetadata(ctx, "https://youtube.com/watch?v=test")
	if err == nil {
		t.Error("Expected error when yt-dlp execution fails")
	}
}

func TestYTDLPDownloader_GetMetadata_InvalidJSON(t *testing.T) {
	tempDir := t.TempDir()
	downloader := NewYTDLPDownloader(tempDir)

	downloader.setCommandRunner(&fakeCommandRunner{
		runFunc: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return []byte("invalid json"), nil
		},
	})

	ctx := context.Background()
	_, err := downloader.GetMetadata(ctx, "https://youtube.com/watch?v=test")
	if err == nil {
		t.Error("Expected error for invalid JSON output")
	}
}

func TestYTDLPDownloader_GetPlaylist_EmptyOutput(t *testing.T) {
	downloader := NewYTDLPDownloader(t.TempDir())
	downloader.setCommandRunner(&fakeCommandRunner{
		runFunc: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return []byte(" \n\t "), nil
		},
	})

	_, err := downloader.GetPlaylist(context.Background(), "https://youtube.com/playlist?list=test")
	if err == nil {
		t.Fatal("Expected error for empty playlist output")
	}
	if !errors.Is(err, ErrNotAPlaylist) {
		t.Errorf("Expected ErrNotAPlaylist, got %v", err)
	}
}

func TestYTDLPDownloader_GetPlaylist_SuccessWithInvalidJSONLines(t *testing.T) {
	downloader := NewYTDLPDownloader(t.TempDir())
	downloader.setCommandRunner(&fakeCommandRunner{
		runFunc: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return []byte(`not-json
{"id":"entry1","title":"First","url":"https://youtube.com/watch?v=entry1"}
{"id":"entry2","title":"Second","url":"https://youtube.com/watch?v=entry2"}`), nil
		},
	})

	playlist, err := downloader.GetPlaylist(context.Background(), "https://youtube.com/playlist?list=test")
	if err != nil {
		t.Fatalf("GetPlaylist failed: %v", err)
	}
	if playlist.ID != "entry1" {
		t.Errorf("Expected playlist ID entry1, got %s", playlist.ID)
	}
	if playlist.Title != "First" {
		t.Errorf("Expected playlist title First, got %s", playlist.Title)
	}
	if len(playlist.Entries) != 2 {
		t.Fatalf("Expected 2 entries, got %d", len(playlist.Entries))
	}
	if playlist.Entries[0].ID != "entry1" || playlist.Entries[1].ID != "entry2" {
		t.Errorf("Unexpected playlist entries: %+v", playlist.Entries)
	}
}

func TestYTDLPDownloader_GetPlaylist_ExecutionError(t *testing.T) {
	downloader := NewYTDLPDownloader(t.TempDir())
	downloader.setCommandRunner(&fakeCommandRunner{
		runFunc: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return nil, errors.New("yt-dlp not found")
		},
	})

	_, err := downloader.GetPlaylist(context.Background(), "https://youtube.com/playlist?list=test")
	if err == nil {
		t.Fatal("Expected error when yt-dlp execution fails")
	}
}

func TestYTDLPDownloader_GetPlaylist_ContextCancelled(t *testing.T) {
	downloader := NewYTDLPDownloader(t.TempDir())

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	downloader.setCommandRunner(&fakeCommandRunner{
		runFunc: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return nil, context.Canceled
		},
	})

	_, err := downloader.GetPlaylist(ctx, "https://youtube.com/playlist?list=test")
	if err == nil {
		t.Fatal("Expected error when context is cancelled")
	}
	if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Expected context cancellation error, got: %v", err)
	}
}

func TestYTDLPDownloader_DownloadWithContext_ContextCancelled(t *testing.T) {
	tempDir := t.TempDir()
	downloader := NewYTDLPDownloader(tempDir)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	downloader.setCommandRunner(&fakeCommandRunner{
		combinedOutputFunc: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return nil, context.Canceled
		},
	})

	_, err := downloader.DownloadWithContext(ctx, Item{ID: "test", URL: "https://youtube.com/watch?v=test"})
	if err == nil {
		t.Error("Expected error when context is cancelled")
	}
}

func TestYTDLPDownloader_DownloadWithContext_EmptyURL(t *testing.T) {
	tempDir := t.TempDir()
	downloader := NewYTDLPDownloader(tempDir)

	ctx := context.Background()
	_, err := downloader.DownloadWithContext(ctx, Item{ID: "test", URL: ""})
	if err == nil {
		t.Error("Expected error for empty URL")
	}
}

func TestYTDLPDownloader_DownloadWithContext_Success(t *testing.T) {
	tempDir := t.TempDir()
	downloader := NewYTDLPDownloader(tempDir)

	infoJSON := `{"id": "test123", "title": "Test Video", "duration": 120, "format": "mp4", "ext": "mp4", "filesize": 1024000}`
	infoPath := filepath.Join(tempDir, "test123.info.json")
	if err := os.WriteFile(infoPath, []byte(infoJSON), 0644); err != nil {
		t.Fatalf("Failed to write test info.json: %v", err)
	}

	videoPath := filepath.Join(tempDir, "test123.mp4")
	if err := os.WriteFile(videoPath, []byte("fake video content"), 0644); err != nil {
		t.Fatalf("Failed to write test video file: %v", err)
	}

	downloader.setCommandRunner(&fakeCommandRunner{
		combinedOutputFunc: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			return []byte(""), nil
		},
	})

	ctx := context.Background()
	result, err := downloader.DownloadWithContext(ctx, Item{ID: "test123", URL: "https://youtube.com/watch?v=test123"})
	if err != nil {
		t.Fatalf("DownloadWithContext failed: %v", err)
	}

	if result.ID != "test123" {
		t.Errorf("Expected ID test123, got %s", result.ID)
	}
	if result.FilePath != videoPath {
		t.Errorf("Expected FilePath %s, got %s", videoPath, result.FilePath)
	}
}
