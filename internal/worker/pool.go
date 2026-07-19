package worker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"streamforge/internal/downloader"
	"streamforge/internal/jobs"
	"streamforge/internal/queue"
	"streamforge/internal/redis"
)

type Pool struct {
	workers    int
	taskQueue  chan queue.Message
	jobService *jobs.Service
	redis      *redis.Client
	downloader downloader.Downloader
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
}

func NewPool(
	workers int,
	jobSvc *jobs.Service,
	redisClient *redis.Client,
	dl downloader.Downloader,
) *Pool {
	ctx, cancel := context.WithCancel(context.Background())
	if dl == nil {
		dl = downloader.NewMockDownloader()
	}
	return &Pool{
		workers:    workers,
		taskQueue:  make(chan queue.Message, workers*2),
		jobService: jobSvc,
		redis:      redisClient,
		downloader: dl,
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

	sourceURL, _ := task.Payload["source_url"].(string)

	_ = p.jobService.UpdateJobStatus(ctx, jobID, "PROCESSING", "")
	if p.redis != nil {
		_ = p.redis.SetJobStatus(ctx, jobID, "PROCESSING")
	}

	items, _, err := p.jobService.GetMediaItemsInternal(ctx, jobID, "", 100, 0)
	if err != nil {
		p.failJob(ctx, jobID, err.Error())
		return
	}

	if len(items) == 0 {
		if sourceURL == "" {
			p.failJob(ctx, jobID, "source_url is required")
			return
		}
		if err := p.jobService.CreateMediaItems(ctx, jobID, []jobs.MediaItemInput{
			{Title: "Source Media", SourceURL: sourceURL},
		}); err != nil {
			p.failJob(ctx, jobID, err.Error())
			return
		}
		items, _, err = p.jobService.GetMediaItemsInternal(ctx, jobID, "", 100, 0)
		if err != nil {
			p.failJob(ctx, jobID, err.Error())
			return
		}
	}

	totalItems := len(items)
	_ = p.jobService.SetJobTotalItems(ctx, jobID, totalItems)
	if p.redis != nil {
		_ = p.redis.SetProgress(ctx, jobID, &redis.Progress{
			Total:      totalItems,
			Completed:  0,
			Percentage: 0,
			Status:     "PROCESSING",
		})
	}

	completed := 0
	for _, item := range items {
		select {
		case <-ctx.Done():
			return
		default:
		}

		_ = p.jobService.UpdateMediaItemStatus(ctx, item.ID.String(), "PROCESSING", 0, "")
		result, err := p.downloader.Download(downloader.Item{
			ID:    item.ID.String(),
			URL:   item.SourceURL,
			Title: item.Title,
		})
		if err != nil {
			_ = p.jobService.UpdateMediaItemStatus(ctx, item.ID.String(), "FAILED", 0, err.Error())
			p.failJob(ctx, jobID, err.Error())
			return
		}

		_ = p.jobService.UpdateMediaItemStatus(ctx, item.ID.String(), "COMPLETED", 100, "")
		_ = p.jobService.UpdateMediaItemSize(ctx, item.ID.String(), result.Size)

		completed++
		_ = p.jobService.UpdateJobProgress(ctx, jobID, completed)
	}

	_ = p.jobService.UpdateJobStatus(ctx, jobID, "COMPLETED", "")
	if p.redis != nil {
		_ = p.redis.SetJobStatus(ctx, jobID, "COMPLETED")
		_ = p.redis.PublishJobEvent(ctx, jobID, "complete", map[string]string{"status": "COMPLETED"})
	}
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

func (p *Pool) failJob(ctx context.Context, jobID, message string) {
	_ = p.jobService.UpdateJobStatus(ctx, jobID, "FAILED", message)
	if p.redis != nil {
		_ = p.redis.SetJobStatus(ctx, jobID, "FAILED")
		_ = p.redis.PublishJobEvent(ctx, jobID, "failed", map[string]string{
			"status":  "FAILED",
			"message": message,
		})
	}
}
