package worker

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"streamforge/internal/database"
	"streamforge/internal/downloader"
	"streamforge/internal/queue"
)

type mockJobService struct {
	items                 []*database.MediaItem
	getItemsErr           error
	statusUpdates         []string
	itemStatusUpdates     []string
	sizeUpdates           []int64
	progressUpdates       []int
	setTotalItemsRequests []int
}

func (m *mockJobService) UpdateJobStatus(ctx context.Context, jobID, status, errorMsg string) error {
	m.statusUpdates = append(m.statusUpdates, status)
	return nil
}

func (m *mockJobService) GetMediaItemsInternal(ctx context.Context, jobID, status string, limit, offset int) ([]*database.MediaItem, int, error) {
	if m.getItemsErr != nil {
		return nil, 0, m.getItemsErr
	}
	return m.items, len(m.items), nil
}

func (m *mockJobService) SetJobTotalItems(ctx context.Context, jobID string, total int) error {
	m.setTotalItemsRequests = append(m.setTotalItemsRequests, total)
	return nil
}

func (m *mockJobService) UpdateMediaItemStatus(ctx context.Context, itemID, status string, progress int, errorMsg string) error {
	m.itemStatusUpdates = append(m.itemStatusUpdates, status)
	return nil
}

func (m *mockJobService) UpdateMediaItemSize(ctx context.Context, itemID string, size int64) error {
	m.sizeUpdates = append(m.sizeUpdates, size)
	return nil
}

func (m *mockJobService) UpdateJobProgress(ctx context.Context, jobID string, completed int) error {
	m.progressUpdates = append(m.progressUpdates, completed)
	return nil
}

type mockDownloader struct {
	result downloader.Result
	err    error
}

func (m *mockDownloader) Download(item downloader.Item) (downloader.Result, error) {
	if m.err != nil {
		return downloader.Result{}, m.err
	}
	return m.result, nil
}

func TestNewPool(t *testing.T) {
	jobSvc := &mockJobService{}

	pool := NewPool(4, jobSvc, nil, downloader.NewMockDownloader())
	assert.NotNil(t, pool)
	assert.Equal(t, 4, pool.workers)
	assert.Equal(t, 8, cap(pool.taskQueue))
}

func TestPool_SubmitWithoutStart(t *testing.T) {
	jobSvc := &mockJobService{}

	pool := NewPool(1, jobSvc, nil, downloader.NewMockDownloader())
	err := pool.Submit("job-123", "https://example.com/playlist.m3u8")
	assert.NoError(t, err)
}

func TestPool_Submit_QueueFull(t *testing.T) {
	jobSvc := &mockJobService{}

	pool := NewPool(1, jobSvc, nil, downloader.NewMockDownloader())
	pool.taskQueue = make(chan queue.Message, 1)

	pool.taskQueue <- queue.Message{JobID: "job-1"}

	err := pool.Submit("job-2", "https://example.com/playlist.m3u8")
	assert.Error(t, err)
	assert.Equal(t, "queue full", err.Error())
}

func TestPool_JobChan(t *testing.T) {
	jobSvc := &mockJobService{}

	pool := NewPool(1, jobSvc, nil, downloader.NewMockDownloader())
	ch := pool.JobChan()
	assert.NotNil(t, ch)
	assert.IsType(t, make(chan<- queue.Message, 2), ch)
}

func TestPool_ProcessTask_NoMediaItems_FailsJob(t *testing.T) {
	jobSvc := &mockJobService{items: []*database.MediaItem{}}
	pool := NewPool(1, jobSvc, nil, &mockDownloader{})

	pool.processTask(queue.Message{JobID: "job-1"})

	assert.Equal(t, []string{"PROCESSING", "FAILED"}, jobSvc.statusUpdates)
	assert.Empty(t, jobSvc.itemStatusUpdates)
}

func TestPool_ProcessTask_DownloadFailure(t *testing.T) {
	itemID := uuid.New()
	jobSvc := &mockJobService{
		items: []*database.MediaItem{
			{ID: itemID, SourceURL: "https://example.com/video.mp4", Title: "video"},
		},
	}
	pool := NewPool(1, jobSvc, nil, &mockDownloader{err: errors.New("download failed")})

	pool.processTask(queue.Message{JobID: "job-2"})

	assert.Equal(t, []string{"PROCESSING", "FAILED"}, jobSvc.statusUpdates)
	assert.Equal(t, []string{"PROCESSING", "FAILED"}, jobSvc.itemStatusUpdates)
}

func TestPool_ProcessTask_Success(t *testing.T) {
	itemID := uuid.New()
	jobSvc := &mockJobService{
		items: []*database.MediaItem{
			{ID: itemID, SourceURL: "https://example.com/video.mp4", Title: "video"},
		},
	}
	pool := NewPool(1, jobSvc, nil, &mockDownloader{
		result: downloader.Result{ID: itemID.String(), Size: 2048},
	})

	pool.processTask(queue.Message{JobID: "job-3"})

	assert.Equal(t, []string{"PROCESSING", "COMPLETED"}, jobSvc.statusUpdates)
	assert.Equal(t, []string{"PROCESSING", "COMPLETED"}, jobSvc.itemStatusUpdates)
	assert.Equal(t, []int64{2048}, jobSvc.sizeUpdates)
	assert.Equal(t, []int{1}, jobSvc.progressUpdates)
	assert.Equal(t, []int{1}, jobSvc.setTotalItemsRequests)
}
