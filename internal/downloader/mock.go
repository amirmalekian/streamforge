package downloader

import (
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
