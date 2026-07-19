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

func TestCreateJobHandler_PublishFailureMarksFailed(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &fakeJobCreateService{jobID: uuid.New()}
	queueSvc := &fakeQueuePublisher{err: errors.New("publish failed")}
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", uuid.New().String())
		c.Next()
	})
	router.POST("/jobs", createJobHandler(svc, queueSvc))

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
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", uuid.New().String())
		c.Next()
	})
	router.POST("/jobs", createJobHandler(svc, queueSvc))

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
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set("user_id", uuid.New().String())
		c.Next()
	})
	router.POST("/jobs", createJobHandler(svc, queueSvc))

	body, _ := json.Marshal(map[string]string{"source_url": "https://example.com/video.mp4"})
	req := httptest.NewRequest(http.MethodPost, "/jobs", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Equal(t, []string{"FAILED"}, svc.statusUpdates)
	assert.Empty(t, queueSvc.messages)
}
