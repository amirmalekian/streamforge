package jobs

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"streamforge/internal/database"
	"streamforge/internal/redis"
)

func TestNewService(t *testing.T) {
	repo := &database.Repository{}
	redisClient := &redis.Client{}

	svc := NewService(repo, redisClient)
	assert.NotNil(t, svc)
	assert.Equal(t, repo, svc.repo)
	assert.Equal(t, redisClient, svc.redis)
}

func TestJobToResponse(t *testing.T) {
	job := &database.Job{
		ID:             uuid.New(),
		UserID:         uuid.New(),
		SourceURL:      "https://example.com/playlist.m3u8",
		Status:         "PROCESSING",
		TotalItems:     10,
		CompletedItems: 5,
		ErrorMessage:   "",
		CreatedAt:      time.Now().Format(time.RFC3339),
		UpdatedAt:      time.Now().Format(time.RFC3339),
	}

	resp := jobToResponse(job)

	assert.Equal(t, job.ID, resp.ID)
	assert.Equal(t, job.UserID, resp.UserID)
	assert.Equal(t, job.SourceURL, resp.SourceURL)
	assert.Equal(t, job.Status, resp.Status)
	assert.Equal(t, job.TotalItems, resp.TotalItems)
	assert.Equal(t, job.CompletedItems, resp.CompletedItems)
	assert.Equal(t, 50, resp.ProgressPercent)
	assert.Equal(t, job.CreatedAt, resp.CreatedAt)
	assert.Equal(t, job.UpdatedAt, resp.UpdatedAt)
}

func TestJobToResponse_ZeroTotal(t *testing.T) {
	job := &database.Job{
		ID:             uuid.New(),
		UserID:         uuid.New(),
		SourceURL:      "https://example.com/playlist.m3u8",
		Status:         "CREATED",
		TotalItems:     0,
		CompletedItems: 0,
	}

	resp := jobToResponse(job)
	assert.Equal(t, 0, resp.ProgressPercent)
}

func TestMediaItemToResponse(t *testing.T) {
	item := &database.MediaItem{
		ID:           uuid.New(),
		JobID:        uuid.New(),
		Title:        "Test Media",
		SourceURL:    "https://example.com/media.mp4",
		Status:       "COMPLETED",
		Progress:     100,
		SizeBytes:    1024,
		ErrorMessage: "",
		CreatedAt:    time.Now().Format(time.RFC3339),
		UpdatedAt:    time.Now().Format(time.RFC3339),
	}

	resp := mediaItemToResponse(item)

	assert.Equal(t, item.ID, resp.ID)
	assert.Equal(t, item.JobID, resp.JobID)
	assert.Equal(t, item.Title, resp.Title)
	assert.Equal(t, item.SourceURL, resp.SourceURL)
	assert.Equal(t, item.Status, resp.Status)
	assert.Equal(t, item.Progress, resp.Progress)
	assert.Equal(t, item.SizeBytes, resp.SizeBytes)
	assert.Equal(t, item.CreatedAt, resp.CreatedAt)
	assert.Equal(t, item.UpdatedAt, resp.UpdatedAt)
}

func TestErrJobNotFound(t *testing.T) {
	assert.Equal(t, "job not found", ErrJobNotFound.Error())
}

func TestErrUnauthorized(t *testing.T) {
	assert.Equal(t, "unauthorized", ErrUnauthorized.Error())
}

func TestErrCannotCancel(t *testing.T) {
	assert.Equal(t, "cannot cancel job", ErrCannotCancel.Error())
}
