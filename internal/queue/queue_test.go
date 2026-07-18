package queue

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMessage_JSON(t *testing.T) {
	msg := Message{
		JobID:     "job-123",
		Action:    "PROCESS",
		Payload:   map[string]interface{}{"source_url": "https://example.com/playlist.m3u8"},
		CreatedAt: time.Now().Format(time.RFC3339),
	}

	data, err := json.Marshal(msg)
	assert.NoError(t, err)

	var unmarshaled Message
	err = json.Unmarshal(data, &unmarshaled)
	assert.NoError(t, err)
	assert.Equal(t, msg.JobID, unmarshaled.JobID)
	assert.Equal(t, msg.Action, unmarshaled.Action)
	assert.Equal(t, msg.Payload, unmarshaled.Payload)
}

func TestMessage_Empty(t *testing.T) {
	msg := Message{}
	data, err := json.Marshal(msg)
	assert.NoError(t, err)

	var unmarshaled Message
	err = json.Unmarshal(data, &unmarshaled)
	assert.NoError(t, err)
	assert.Empty(t, unmarshaled.JobID)
	assert.Empty(t, unmarshaled.Action)
	assert.Nil(t, unmarshaled.Payload)
}