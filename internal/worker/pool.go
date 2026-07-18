package worker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"streamforge/internal/jobs"
	"streamforge/internal/queue"
	"streamforge/internal/redis"
)

type Pool struct {
	workers    int
	taskQueue  chan queue.Message
	jobService *jobs.Service
	redis      *redis.Client
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
}

func NewPool(
	workers int,
	jobSvc *jobs.Service,
	redisClient *redis.Client,
) *Pool {
	ctx, cancel := context.WithCancel(context.Background())
	return &Pool{
		workers:    workers,
		taskQueue:  make(chan queue.Message, workers*2),
		jobService: jobSvc,
		redis:      redisClient,
		ctx:        ctx,
		cancel:     cancel,
	}
}

func (p *Pool) Start(ctx context.Context) {
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
}

func (p *Pool) worker(id int) {
	defer p.wg.Done()
	for {
		select {
		case task, ok := <-p.taskQueue:
			if !ok {
				return
			}
			p.processTask(task)
		case <-p.ctx.Done():
			return
		}
	}
}

func (p *Pool) processTask(task queue.Message) {
	jobID := task.JobID
	ctx := p.ctx

	p.jobService.UpdateJobStatus(ctx, jobID, "PROCESSING", "")

	progress, _ := p.redis.GetProgress(ctx, jobID)
	if progress == nil {
		progress = &redis.Progress{
			Total:      10,
			Completed:  0,
			Percentage: 0,
			Status:     "PROCESSING",
		}
		p.redis.SetProgress(ctx, jobID, progress)
	}

	items, _, err := p.jobService.GetMediaItems(ctx, "", jobID, "", 100, 0)
	if err != nil {
		p.jobService.UpdateJobStatus(ctx, jobID, "FAILED", err.Error())
		return
	}

	if len(items) == 0 {
		p.jobService.SetJobTotalItems(ctx, jobID, 10)
		for i := 1; i <= 10; i++ {
			p.jobService.CreateMediaItems(ctx, jobID, []jobs.MediaItemInput{
				{Title: fmt.Sprintf("Media Item %d", i), SourceURL: fmt.Sprintf("https://example.com/media/%d.mp4", i)},
			})
		}
		items, _, _ = p.jobService.GetMediaItems(ctx, "", jobID, "", 100, 0)
	}

	for _, item := range items {
		select {
		case <-ctx.Done():
			return
		default:
		}

		p.jobService.UpdateMediaItemStatus(ctx, item.ID.String(), "PROCESSING", 0, "")

		p.simulateProcessing(item.ID)

		p.jobService.UpdateMediaItemStatus(ctx, item.ID.String(), "COMPLETED", 100, "")

		updated, _ := p.redis.IncrementProgress(ctx, jobID)
		if updated != nil {
			p.redis.PublishJobEvent(ctx, jobID, "progress", updated)
			p.jobService.UpdateJobProgress(ctx, jobID, updated.Completed)
		}
	}

	p.jobService.UpdateJobStatus(ctx, jobID, "COMPLETED", "")
	p.redis.SetJobStatus(ctx, jobID, "COMPLETED")
	p.redis.PublishJobEvent(ctx, jobID, "complete", map[string]string{"status": "COMPLETED"})
}

func (p *Pool) simulateProcessing(itemID uuid.UUID) {
	time.Sleep(100 * time.Millisecond)
}

func (p *Pool) Submit(jobID, sourceURL string) error {
	msg := queue.Message{
		JobID:     jobID,
		Action:    "PROCESS",
		Payload:   map[string]interface{}{"source_url": sourceURL},
		CreatedAt: time.Now().Format(time.RFC3339),
	}

	select {
	case p.taskQueue <- msg:
		return nil
	case <-p.ctx.Done():
		return fmt.Errorf("pool closed")
	default:
		return fmt.Errorf("queue full")
	}
}

func (p *Pool) JobChan() chan<- queue.Message {
	return p.taskQueue
}

func (p *Pool) Stop() {
	p.cancel()
	close(p.taskQueue)
	p.wg.Wait()
}
