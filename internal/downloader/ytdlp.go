package downloader

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type CommandRunner interface {
	Run(ctx context.Context, name string, args ...string) ([]byte, error)
	CombinedOutput(ctx context.Context, name string, args ...string) ([]byte, error)
}

type realCommandRunner struct{}

func (r *realCommandRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	return cmd.Output()
}

func (r *realCommandRunner) CombinedOutput(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	return cmd.CombinedOutput()
}

type YTDLPDownloader struct {
	outputDir string
	timeout   time.Duration
	cmdRunner CommandRunner
}

func NewYTDLPDownloader(outputDir string) *YTDLPDownloader {
	if outputDir == "" {
		outputDir = "/tmp"
	}
	return &YTDLPDownloader{
		outputDir: outputDir,
		timeout:   30 * time.Minute,
		cmdRunner: &realCommandRunner{},
	}
}

func (d *YTDLPDownloader) setCommandRunner(runner CommandRunner) {
	d.cmdRunner = runner
}

func (d *YTDLPDownloader) Download(item Item) (Result, error) {
	ctx, cancel := context.WithTimeout(context.Background(), d.timeout)
	defer cancel()
	return d.DownloadWithContext(ctx, item)
}

func (d *YTDLPDownloader) DownloadWithContext(ctx context.Context, item Item) (Result, error) {
	if item.URL == "" {
		return Result{}, fmt.Errorf("url is required")
	}

	outputPath := filepath.Join(d.outputDir, fmt.Sprintf("%s.%%(ext)s", item.ID))
	if item.ID == "" {
		outputPath = filepath.Join(d.outputDir, "%(id)s.%%(ext)s")
	}

	args := []string{
		"--no-playlist",
		"--write-info-json",
		"--output", outputPath,
		item.URL,
	}

	output, err := d.cmdRunner.CombinedOutput(ctx, "yt-dlp", args...)
	if err != nil {
		if ctx.Err() == context.Canceled || ctx.Err() == context.DeadlineExceeded {
			return Result{}, fmt.Errorf("download cancelled: %w", ctx.Err())
		}
		return Result{}, fmt.Errorf("yt-dlp failed: %s: %w", string(output), err)
	}

	result, err := d.parseResult(item.ID)
	if err != nil {
		return Result{}, fmt.Errorf("failed to parse result: %w", err)
	}

	return result, nil
}

func (d *YTDLPDownloader) parseResult(expectedID string) (Result, error) {
	// TODO: This method scans the output directory which is not safe for concurrent downloads.
	// Consider using a unique temp directory per download or returning the file path directly from yt-dlp.
	entries, err := os.ReadDir(d.outputDir)
	if err != nil {
		return Result{}, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".info.json") {
			infoPath := filepath.Join(d.outputDir, name)
			data, err := os.ReadFile(infoPath)
			if err != nil {
				continue
			}

			var info YTDLPInfo
			if err := json.Unmarshal(data, &info); err != nil {
				continue
			}

			if expectedID != "" && info.ID != expectedID {
				continue
			}

			filePath := strings.TrimSuffix(infoPath, ".info.json")
			if _, err := os.Stat(filePath); err != nil {
				for _, ext := range []string{".mp4", ".webm", ".mkv", ".mov", ".flv"} {
					testPath := filePath + ext
					if _, err := os.Stat(testPath); err == nil {
						filePath = testPath
						break
					}
				}
			}

			if _, err := os.Stat(filePath); err != nil {
				return Result{}, fmt.Errorf("video file not found for %s", info.ID)
			}

			fileInfo, err := os.Stat(filePath)
			size := int64(0)
			if err == nil {
				size = fileInfo.Size()
			}

			return Result{
				ID:       info.ID,
				FilePath: filePath,
				Size:     size,
			}, nil
		}
	}

	return Result{}, fmt.Errorf("downloaded file not found")
}

type YTDLPInfo struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Duration int    `json:"duration"`
	Format   string `json:"format"`
	Ext      string `json:"ext"`
	FileSize int64  `json:"filesize"`
}

func (d *YTDLPDownloader) GetMetadata(ctx context.Context, url string) (*Metadata, error) {
	args := []string{
		"--dump-json",
		"--no-playlist",
		url,
	}

	output, err := d.cmdRunner.Run(ctx, "yt-dlp", args...)
	if err != nil {
		if ctx.Err() == context.Canceled || ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("metadata fetch cancelled: %w", ctx.Err())
		}
		return nil, fmt.Errorf("yt-dlp metadata failed: %w", err)
	}

	var info YTDLPInfo
	if err := json.Unmarshal(output, &info); err != nil {
		return nil, fmt.Errorf("failed to parse metadata: %w", err)
	}

	return &Metadata{
		ID:       info.ID,
		Title:    info.Title,
		Duration: info.Duration,
		Format:   info.Format,
		Ext:      info.Ext,
		Size:     info.FileSize,
	}, nil
}

func (d *YTDLPDownloader) GetPlaylist(ctx context.Context, url string) (*Playlist, error) {
	args := []string{
		"--flat-playlist",
		"--dump-json",
		url,
	}

	output, err := d.cmdRunner.Run(ctx, "yt-dlp", args...)
	if err != nil {
		if ctx.Err() == context.Canceled || ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("playlist fetch cancelled: %w", ctx.Err())
		}
		return nil, fmt.Errorf("yt-dlp playlist failed: %w", err)
	}

trimmed := strings.TrimSpace(string(output))
if trimmed == "" {
	return nil, fmt.Errorf("no playlist entries found")
}
lines := strings.Split(trimmed, "\n")

	var entries []PlaylistEntry
	var playlistID, playlistTitle string

	for i, line := range lines {
		var info YTDLPPlaylistInfo
		if err := json.Unmarshal([]byte(line), &info); err != nil {
			continue
		}

		if playlistID == "" && playlistTitle == "" {
			playlistID = info.ID
			playlistTitle = info.Title
		}

		if info.ID != "" && info.URL != "" {
			entries = append(entries, PlaylistEntry{
				ID:    info.ID,
				URL:   info.URL,
				Title: info.Title,
			})
		}
	}

	if len(entries) == 0 {
		return nil, fmt.Errorf("no valid playlist entries found")
	}

	return &Playlist{
		ID:      playlistID,
		Title:   playlistTitle,
		Entries: entries,
	}, nil
}

type YTDLPPlaylistInfo struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	URL   string `json:"url"`
}

func (d *YTDLPDownloader) SetTimeout(timeout time.Duration) {
	d.timeout = timeout
}
