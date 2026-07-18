package worker

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"streamforge/internal/jobs"
	"streamforge/internal/queue"
	"streamforge/internal/redis"
)

func TestNewPool(t *testing.T) {
	jobSvc := &jobs.Service{}
	redisClient := &redis.Client{}

	pool := NewPool(4, jobSvc, redisClient)
	assert.NotNil(t, pool)
	assert.Equal(t, 4, pool.workers)
	assert.Equal(t, 8, cap(pool.taskQueue))
}

func TestPool_SubmitWithoutStart(t *testing.T) {
	jobSvc := &jobs.Service{}
	redisClient := &redis.Client{}

	pool := NewPool(1, jobSvc, redisClient)
	err := pool.Submit("job-123", "https://example.com/playlist.m3u8")
	assert.NoError(t, err)
}

func TestPool_Submit_QueueFull(t *testing.T) {
	jobSvc := &jobs.Service{}
	redisClient := &redis.Client{}

	pool := NewPool(1, jobSvc, redisClient)
	pool.taskQueue = make(chan queue.Message, 1)

	pool.taskQueue <- queue.Message{JobID: "job-1"}

	err := pool.Submit("job-2", "https://example.com/playlist.m3u8")
	assert.Error(t, err)
	assert.Equal(t, "queue full", err.Error())
}

func TestPool_JobChan(t *testing.T) {
	jobSvc := &jobs.Service{}
	redisClient := &redis.Client{}

	pool := NewPool(1, jobSvc, redisClient)
	ch := pool.JobChan()
	assert.NotNil(t, ch)
	assert.IsType(t, make(chan<- queue.Message, 2), ch)
}