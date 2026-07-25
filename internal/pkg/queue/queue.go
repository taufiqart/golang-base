package queue

import (
	"context"
	"time"
)

type Task struct {
	Type    string
	Payload []byte
}

type EnqueueOptions struct {
	ProcessIn time.Duration
	Queue     string
	MaxRetry  int
}

type Client interface {
	Enqueue(ctx context.Context, task *Task, opts ...EnqueueOptions) error
	Close() error
}
