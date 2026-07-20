package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"streamforge/internal/downloader"
	"streamforge/internal/jobs"
	"streamforge/internal/queue"
)

type fakeJobCreateService struct {
	jobID          uuid.UUID
	createErr      error
	createItemsErr error
	statusUpdates  []string
	createdItems   []jobs.MediaItemInput
}

func (f *fakeJobCreateService) CreateJob(ctx context.Context, userID, sourceURL string) (*jobs.JobResponse, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}
	return &jobs.JobResponse{ID: f.jobID}, nil
}

func (f *fakeJobCreateService) CreateMediaItems(ctx context.Context, jobID string, items []jobs.MediaItemInput) error {
	if f.createItemsErr != nil {
		return f.createItemsErr
	}
	f.createdItems = append(f.createdItems, items...)
	return nil
}

func (f *fakeJobCreateService) UpdateJobStatus(ctx context.Context, jobID, status, errorMsg string) error {
	f.statusUpdates = append(f.statusUpdates, status)
	return nil
}

type fakeQueuePublisher struct {
	err      error
	messages []queue.Message
}

func (f *fakeQueuePublisher) Publish(ctx context.Context, msg queue.Message) error {
	if f.err != nil {
		return f.err
	}
	f.messages = append(f.messages, msg)
	return nil
}

type fakeDownloader struct {
	playlist *downloader.Playlist
	err      error
}

func (f *fakeDownloader) Download(item downloader.Item) (downloader.Result, error) {
	return downloader.Result{}, nil
}

func (f *fakeDownloader) GetPlaylist(ctx context.Context, url string) (*downloader.Playlist, error) {
	return f.playlist, f.err
}

func TestCreateJobHandler_PublishFailureMarksFailed(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &fakeJobCreateService{jobID: uuid.New()}
	queueSvc := &fakeQueuePublisher{err: errors.New("publish failed")}
	dl := &fakeDownloader{}
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", uuid.New().String())
		c.Next()
	})
	router.POST("/jobs", createJobHandler(svc, queueSvc, dl))

	body, _ := json.Marshal(map[string]string{"source_url": "https://example.com/video.mp4"})
	req := httptest.NewRequest(http.MethodPost, "/jobs", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Equal(t, []string{"FAILED"}, svc.statusUpdates)
	assert.Len(t, svc.createdItems, 1)
}

func TestCreateJobHandler_SuccessMarksQueuedAndPublishes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &fakeJobCreateService{jobID: uuid.New()}
	queueSvc := &fakeQueuePublisher{}
	dl := &fakeDownloader{}
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", uuid.New().String())
		c.Next()
	})
	router.POST("/jobs", createJobHandler(svc, queueSvc, dl))

	body, _ := json.Marshal(map[string]string{"source_url": "https://example.com/video.mp4"})
	req := httptest.NewRequest(http.MethodPost, "/jobs", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)
	assert.Equal(t, []string{"QUEUED"}, svc.statusUpdates)
	assert.Len(t, queueSvc.messages, 1)
	assert.Equal(t, "PROCESS", queueSvc.messages[0].Action)
	assert.Equal(t, "https://example.com/video.mp4", queueSvc.messages[0].Payload["source_url"])
}

func TestCreateJobHandler_CreateMediaItemsFailureMarksFailed(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &fakeJobCreateService{
		jobID:          uuid.New(),
		createItemsErr: errors.New("media create failed"),
	}
	queueSvc := &fakeQueuePublisher{}
	dl := &fakeDownloader{}
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", uuid.New().String())
		c.Next()
	})
	router.POST("/jobs", createJobHandler(svc, queueSvc, dl))

	body, _ := json.Marshal(map[string]string{"source_url": "https://example.com/video.mp4"})
	req := httptest.NewRequest(http.MethodPost, "/jobs", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Equal(t, []string{"FAILED"}, svc.statusUpdates)
	assert.Empty(t, queueSvc.messages)
}

func TestCreateJobHandler_PlaylistCreatesMultipleItems(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &fakeJobCreateService{jobID: uuid.New()}
	queueSvc := &fakeQueuePublisher{}
	playlist := &downloader.Playlist{
		ID:    "playlist-123",
		Title: "Test Playlist",
		Entries: []downloader.PlaylistEntry{
			{ID: "video1", URL: "https://example.com/video1", Title: "Video 1"},
			{ID: "video2", URL: "https://example.com/video2", Title: "Video 2"},
			{ID: "video3", URL: "https://example.com/video3", Title: "Video 3"},
		},
	}
	dl := &fakeDownloader{playlist: playlist}
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", uuid.New().String())
		c.Next()
	})
	router.POST("/jobs", createJobHandler(svc, queueSvc, dl))

	body, _ := json.Marshal(map[string]string{"source_url": "https://example.com/playlist"})
	req := httptest.NewRequest(http.MethodPost, "/jobs", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)
	assert.Equal(t, []string{"QUEUED"}, svc.statusUpdates)
	assert.Len(t, svc.createdItems, 3)
	assert.Equal(t, "Video 1", svc.createdItems[0].Title)
	assert.Equal(t, "https://example.com/video1", svc.createdItems[0].SourceURL)
	assert.Equal(t, "Video 2", svc.createdItems[1].Title)
	assert.Equal(t, "https://example.com/video2", svc.createdItems[1].SourceURL)
	assert.Equal(t, "Video 3", svc.createdItems[2].Title)
	assert.Equal(t, "https://example.com/video3", svc.createdItems[2].SourceURL)
	assert.Len(t, queueSvc.messages, 1)
}
