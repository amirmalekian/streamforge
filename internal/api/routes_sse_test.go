package api

import (
	"context"
	"testing"
	"time"

	redisv9 "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"

	redis "streamforge/internal/redis"
)

func TestCurrentProgress_NilRedis_ReturnsFalse(t *testing.T) {
	progress, ok := currentProgress(nil, context.Background(), "job-1")
	assert.False(t, ok)
	assert.Nil(t, progress)
}

func TestCurrentProgress_NoProgressStored_ReturnsFalse(t *testing.T) {
	client := &mockProgressGetter{progress: make(map[string]*redis.Progress)}
	progress, ok := currentProgress(client, context.Background(), "job-1")
	assert.False(t, ok)
	assert.Nil(t, progress)
}

func TestCurrentProgress_ProgressExists_ReturnsTrue(t *testing.T) {
	expected := &redis.Progress{
		Total:      5,
		Completed:  2,
		Percentage: 40,
		Status:     "PROCESSING",
		UpdatedAt:  time.Now().Format(time.RFC3339),
	}
	client := &mockProgressGetter{progress: map[string]*redis.Progress{"job-1": expected}}

	progress, ok := currentProgress(client, context.Background(), "job-1")

	assert.True(t, ok)
	assert.Equal(t, expected, progress)
}

// mockProgressGetter implements progressGetter for testing
type mockProgressGetter struct {
	progress map[string]*redis.Progress
}

func (m *mockProgressGetter) GetProgress(ctx context.Context, jobID string) (*redis.Progress, error) {
	if m.progress == nil {
		return nil, nil
	}
	return m.progress[jobID], nil
}

func TestForwardRedisEvents_ForwardsPayload(t *testing.T) {
	in := make(chan *redisv9.Message, 2)
	out := make(chan string, 2)

	in <- &redisv9.Message{Payload: `{"type":"progress","data":{"completed":1}}`}
	in <- &redisv9.Message{Payload: `{"type":"progress","data":{"completed":2}}`}
	close(in)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	forwardRedisEvents(ctx, in, out)

	assert.Len(t, out, 2)
	msg1 := <-out
	msg2 := <-out
	assert.Equal(t, `{"type":"progress","data":{"completed":1}}`, msg1)
	assert.Equal(t, `{"type":"progress","data":{"completed":2}}`, msg2)
}

func TestForwardRedisEvents_ContextCancellation_Stops(t *testing.T) {
	in := make(chan *redisv9.Message, 1)
	out := make(chan string, 1)

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		// Send after context is cancelled
		time.Sleep(10 * time.Millisecond)
		in <- &redisv9.Message{Payload: `{"type":"progress"}`}
	}()

	cancel()
	forwardRedisEvents(ctx, in, out)

	// No message should have been forwarded because context was cancelled
	assert.Empty(t, out)
}

func TestForwardRedisEvents_ChannelClosed_Stops(t *testing.T) {
	in := make(chan *redisv9.Message, 1)
	out := make(chan string, 1)

	close(in)

	forwardRedisEvents(context.Background(), in, out)

	assert.Empty(t, out)
}

func TestForwardRedisEvents_NilMessage_Skipped(t *testing.T) {
	in := make(chan *redisv9.Message, 1)
	out := make(chan string, 1)

	in <- nil
	close(in)

	forwardRedisEvents(context.Background(), in, out)

	assert.Empty(t, out)
}
