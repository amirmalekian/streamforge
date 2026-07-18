package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Client struct {
	*redis.Client
}

type Progress struct {
	Total      int    `json:"total"`
	Completed  int    `json:"completed"`
	Percentage int    `json:"percentage"`
	Status     string `json:"status"`
	UpdatedAt  string `json:"updated_at"`
}

func Connect(addr, password string, db int) *Client {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		panic(fmt.Sprintf("Failed to connect to Redis: %v", err))
	}

	return &Client{Client: client}
}

func (c *Client) SetProgress(ctx context.Context, jobID string, progress *Progress) error {
	key := fmt.Sprintf("job:%s:progress", jobID)
	progress.UpdatedAt = time.Now().Format(time.RFC3339)

	data, err := json.Marshal(progress)
	if err != nil {
		return err
	}

	return c.Client.Set(ctx, key, data, 24*time.Hour).Err()
}

func (c *Client) GetProgress(ctx context.Context, jobID string) (*Progress, error) {
	key := fmt.Sprintf("job:%s:progress", jobID)
	data, err := c.Client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var progress Progress
	if err := json.Unmarshal(data, &progress); err != nil {
		return nil, err
	}
	return &progress, nil
}

func (c *Client) IncrementProgress(ctx context.Context, jobID string) (*Progress, error) {
	key := fmt.Sprintf("job:%s:progress", jobID)

	for {
		data, err := c.Client.Get(ctx, key).Bytes()
		if err == redis.Nil {
			return nil, fmt.Errorf("progress not found")
		}
		if err != nil {
			return nil, err
		}

		var progress Progress
		if err := json.Unmarshal(data, &progress); err != nil {
			return nil, err
		}

		progress.Completed++
		if progress.Total > 0 {
			progress.Percentage = (progress.Completed * 100) / progress.Total
		}
		progress.UpdatedAt = time.Now().Format(time.RFC3339)

		newData, err := json.Marshal(progress)
		if err != nil {
			return nil, err
		}

		txf := c.TxPipeline()
		txf.Set(ctx, key, newData, 24*time.Hour)

		_, err = txf.Exec(ctx)
		if err == nil {
			return &progress, nil
		}

		if err != redis.TxFailedErr {
			return nil, err
		}
	}
}

func (c *Client) SetJobStatus(ctx context.Context, jobID, status string) error {
	key := fmt.Sprintf("job:%s:status", jobID)
	return c.Client.Set(ctx, key, status, 24*time.Hour).Err()
}

func (c *Client) GetJobStatus(ctx context.Context, jobID string) (string, error) {
	key := fmt.Sprintf("job:%s:status", jobID)
	return c.Client.Get(ctx, key).Result()
}

func (c *Client) DeleteProgress(ctx context.Context, jobID string) error {
	keys := []string{
		fmt.Sprintf("job:%s:progress", jobID),
		fmt.Sprintf("job:%s:status", jobID),
	}
	return c.Client.Del(ctx, keys...).Err()
}

func (c *Client) RateLimitCheck(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	rateLimitKey := fmt.Sprintf("ratelimit:%s", key)
	count, err := c.Client.Incr(ctx, rateLimitKey).Result()
	if err != nil {
		return false, err
	}

	if count == 1 {
		c.Expire(ctx, rateLimitKey, window)
	}

	return count <= int64(limit), nil
}

func (c *Client) SubscribeJobEvents(ctx context.Context, jobID string) *redis.PubSub {
	channel := fmt.Sprintf("job:%s:events", jobID)
	return c.Subscribe(ctx, channel)
}

func (c *Client) PublishJobEvent(ctx context.Context, jobID string, eventType string, data interface{}) error {
	channel := fmt.Sprintf("job:%s:events", jobID)
	payload, err := json.Marshal(map[string]interface{}{
		"type":      eventType,
		"job_id":    jobID,
		"data":      data,
		"timestamp": time.Now().Format(time.RFC3339),
	})
	if err != nil {
		return err
	}
	return c.Client.Publish(ctx, channel, payload).Err()
}
