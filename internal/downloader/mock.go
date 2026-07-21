package downloader

import (
	"context"
	"fmt"
	"time"
)

type MockDownloader struct{}

func NewMockDownloader() *MockDownloader {
	return &MockDownloader{}
}

func (d *MockDownloader) Download(item Item) (Result, error) {
	time.Sleep(300 * time.Millisecond)

	return Result{
		ID:       item.ID,
		FilePath: fmt.Sprintf("/tmp/%s.mp4", item.ID),
		Size:     1024,
	}, nil
}

func (d *MockDownloader) GetPlaylist(ctx context.Context, url string) (*Playlist, error) {
	return &Playlist{
		ID:    "test-playlist",
		Title: "Test Playlist",
		Entries: []PlaylistEntry{
			{ID: "video1", URL: "https://example.com/video1", Title: "Video 1"},
			{ID: "video2", URL: "https://example.com/video2", Title: "Video 2"},
		},
	}, nil
}
