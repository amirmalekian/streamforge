package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"sync"
	"time"

	"github.com/google/uuid"

	"streamforge/internal/database"
	"streamforge/internal/redis"
)

var (
	ErrJobNotFound    = errors.New("job not found")
	ErrUnauthorized   = errors.New("unauthorized")
	ErrCannotCancel   = errors.New("cannot cancel job")
	ErrInvalidRequest = errors.New("invalid request")
)

type Service struct {
	repo  *database.Repository
	redis *redis.Client
	mu    sync.RWMutex
	subs  map[string][]chan string
}

type CreateJobRequest struct {
	UserID    string
	SourceURL string
}

type JobResponse struct {
	ID              uuid.UUID `json:"id"`
	UserID          uuid.UUID `json:"user_id"`
	SourceURL       string    `json:"source_url"`
	Status          string    `json:"status"`
	TotalItems      int       `json:"total_items"`
	CompletedItems  int       `json:"completed_items"`
	ProgressPercent int       `json:"progress_percentage"`
	ErrorMessage    string    `json:"error_message,omitempty"`
	CreatedAt       string    `json:"created_at"`
	UpdatedAt       string    `json:"updated_at"`
}

type MediaItemResponse struct {
	ID           uuid.UUID `json:"id"`
	JobID        uuid.UUID `json:"job_id"`
	Title        string    `json:"title"`
	SourceURL    string    `json:"source_url"`
	Status       string    `json:"status"`
	Progress     int       `json:"progress"`
	SizeBytes    int64     `json:"size_bytes"`
	ErrorMessage string    `json:"error_message,omitempty"`
	CreatedAt    string    `json:"created_at"`
	UpdatedAt    string    `json:"updated_at"`
}

type MediaItemInput struct {
	Title     string
	SourceURL string
}

func NewService(repo *database.Repository, redisClient *redis.Client) *Service {
	return &Service{
		repo:  repo,
		redis: redisClient,
		subs:  make(map[string][]chan string),
	}
}

func (s *Service) CreateJob(ctx context.Context, userID, sourceURL string) (*JobResponse, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}

	job, err := s.repo.CreateJob(ctx, userUUID, sourceURL)
	if err != nil {
		return nil, err
	}

	_ = s.redis.SetProgress(ctx, job.ID.String(), &redis.Progress{
		Total:      0,
		Completed:  0,
		Percentage: 0,
		Status:     "CREATED",
	})

	return jobToResponse(job), nil
}

func (s *Service) GetJob(ctx context.Context, userID, jobID string) (*JobResponse, error) {
	job, err := s.repo.GetJob(ctx, jobID)
	if err != nil {
		return nil, err
	}
	if job == nil {
		return nil, ErrJobNotFound
	}

	if job.UserID.String() != userID {
		return nil, ErrUnauthorized
	}

	progress, _ := s.redis.GetProgress(ctx, jobID)
	if progress != nil {
		job.CompletedItems = progress.Completed
		job.TotalItems = progress.Total
	}

	return jobToResponse(job), nil
}

func (s *Service) ListJobs(ctx context.Context, userID, status string, limit, offset int) ([]*JobResponse, int, error) {
	jobs, total, err := s.repo.ListJobs(ctx, userID, status, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]*JobResponse, len(jobs))
	for i, job := range jobs {
		progress, _ := s.redis.GetProgress(ctx, job.ID.String())
		if progress != nil {
			job.CompletedItems = progress.Completed
			job.TotalItems = progress.Total
		}
		responses[i] = jobToResponse(job)
	}

	return responses, total, nil
}

func (s *Service) CancelJob(ctx context.Context, userID, jobID string) error {
	job, err := s.repo.GetJob(ctx, jobID)
	if err != nil {
		return err
	}
	if job == nil {
		return ErrJobNotFound
	}

	if job.UserID.String() != userID {
		return ErrUnauthorized
	}

	terminalStates := map[string]bool{
		"COMPLETED": true,
		"FAILED":    true,
		"CANCELLED": true,
	}
	if terminalStates[job.Status] {
		return ErrCannotCancel
	}

	if err := s.repo.UpdateJobStatus(ctx, jobID, "CANCELLED", "Cancelled by user"); err != nil {
		return err
	}

	_ = s.redis.SetJobStatus(ctx, jobID, "CANCELLED")
	_ = s.redis.DeleteProgress(ctx, jobID)
	_ = s.repo.UpdateMediaItemsStatus(ctx, jobID, "CANCELLED")

	return nil
}

func (s *Service) UpdateJobStatus(ctx context.Context, jobID, status, errorMsg string) error {
	return s.repo.UpdateJobStatus(ctx, jobID, status, errorMsg)
}

func (s *Service) UpdateJobProgress(ctx context.Context, jobID string, completed int) error {
	if err := s.repo.UpdateJobProgress(ctx, jobID, completed); err != nil {
		return err
	}

	if s.redis == nil {
		return nil
	}

	progress, _ := s.redis.GetProgress(ctx, jobID)
	if progress != nil {
		progress.Completed = completed
		if progress.Total > 0 {
			progress.Percentage = (progress.Completed * 100) / progress.Total
		}
		_ = s.redis.SetProgress(ctx, jobID, progress)
		s.PublishEvent(ctx, jobID, "progress", progress)
	}

	return nil
}

func (s *Service) SetJobTotalItems(ctx context.Context, jobID string, total int) error {
	if err := s.repo.SetJobTotalItems(ctx, jobID, total); err != nil {
		return err
	}

	if s.redis != nil {
		progress, _ := s.redis.GetProgress(ctx, jobID)
		if progress != nil {
			progress.Total = total
			if total > 0 {
				progress.Percentage = (progress.Completed * 100) / total
			}
			_ = s.redis.SetProgress(ctx, jobID, progress)
		}
	}

	return nil
}

func (s *Service) GetMediaItems(ctx context.Context, userID, jobID, status string, limit, offset int) ([]*MediaItemResponse, int, error) {
	job, err := s.repo.GetJob(ctx, jobID)
	if err != nil {
		return nil, 0, err
	}
	if job == nil {
		return nil, 0, ErrJobNotFound
	}

	if job.UserID.String() != userID {
		return nil, 0, ErrUnauthorized
	}

	items, total, err := s.repo.ListMediaItems(ctx, jobID, status, limit, offset)
	if err != nil {
		return nil, 0, err
	}

	responses := make([]*MediaItemResponse, len(items))
	for i, item := range items {
		responses[i] = mediaItemToResponse(item)
	}

	return responses, total, nil
}

func (s *Service) GetJobInternal(ctx context.Context, jobID string) (*database.Job, error) {
	job, err := s.repo.GetJob(ctx, jobID)
	if err != nil {
		return nil, err
	}
	if job == nil {
		return nil, ErrJobNotFound
	}
	return job, nil
}

func (s *Service) GetMediaItemsInternal(ctx context.Context, jobID, status string, limit, offset int) ([]*database.MediaItem, int, error) {
	job, err := s.repo.GetJob(ctx, jobID)
	if err != nil {
		return nil, 0, err
	}
	if job == nil {
		return nil, 0, ErrJobNotFound
	}

	return s.repo.ListMediaItems(ctx, jobID, status, limit, offset)
}

func (s *Service) CreateMediaItems(ctx context.Context, jobID string, items []MediaItemInput) error {
	jobUUID, err := uuid.Parse(jobID)
	if err != nil {
		return err
	}

	total := len(items)
	_ = s.SetJobTotalItems(ctx, jobID, total)

	for _, input := range items {
		params := database.CreateMediaItemParams{
			JobID:     jobUUID,
			Title:     input.Title,
			SourceURL: input.SourceURL,
		}
		if _, err := s.repo.CreateMediaItem(ctx, params); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) UpdateMediaItemStatus(ctx context.Context, itemID, status string, progress int, errorMsg string) error {
	if err := s.repo.UpdateMediaItemStatus(ctx, itemID, status, progress, errorMsg); err != nil {
		return err
	}

	s.PublishEvent(ctx, "", "item_update", map[string]interface{}{
		"item_id":  itemID,
		"status":   status,
		"progress": progress,
	})

	return nil
}

func (s *Service) UpdateMediaItemSize(ctx context.Context, itemID string, size int64) error {
	return s.repo.UpdateMediaItemSize(ctx, itemID, size)
}

func (s *Service) Subscribe(ctx context.Context, jobID string, ch chan string) {
	s.mu.Lock()
	s.subs[jobID] = append(s.subs[jobID], ch)
	s.mu.Unlock()

	go func() {
		<-ctx.Done()
		s.mu.Lock()
		if subs, ok := s.subs[jobID]; ok {
			for i, c := range subs {
				if c == ch {
					s.subs[jobID] = append(subs[:i], subs[i+1:]...)
					break
				}
			}
		}
		s.mu.Unlock()
	}()
}

func (s *Service) PublishEvent(ctx context.Context, jobID, eventType string, data interface{}) {
	s.mu.RLock()
	subs := s.subs[jobID]
	s.mu.RUnlock()

	if len(subs) == 0 {
		return
	}

	payload, err := json.Marshal(map[string]interface{}{
		"type":      eventType,
		"job_id":    jobID,
		"data":      data,
		"timestamp": time.Now().Format(time.RFC3339),
	})
	if err != nil {
		return
	}

	for _, ch := range subs {
		select {
		case ch <- string(payload):
		default:
		}
	}

	if s.redis != nil {
		_ = s.redis.PublishJobEvent(ctx, jobID, eventType, data)
	}
}

func jobToResponse(job *database.Job) *JobResponse {
	progress := 0
	if job.TotalItems > 0 {
		progress = int(math.Round(float64(job.CompletedItems) / float64(job.TotalItems) * 100))
	}

	return &JobResponse{
		ID:              job.ID,
		UserID:          job.UserID,
		SourceURL:       job.SourceURL,
		Status:          job.Status,
		TotalItems:      job.TotalItems,
		CompletedItems:  job.CompletedItems,
		ProgressPercent: progress,
		ErrorMessage:    job.ErrorMessage,
		CreatedAt:       job.CreatedAt,
		UpdatedAt:       job.UpdatedAt,
	}
}

func mediaItemToResponse(item *database.MediaItem) *MediaItemResponse {
	return &MediaItemResponse{
		ID:           item.ID,
		JobID:        item.JobID,
		Title:        item.Title,
		SourceURL:    item.SourceURL,
		Status:       item.Status,
		Progress:     item.Progress,
		SizeBytes:    item.SizeBytes,
		ErrorMessage: item.ErrorMessage,
		CreatedAt:    item.CreatedAt,
		UpdatedAt:    item.UpdatedAt,
	}
}
