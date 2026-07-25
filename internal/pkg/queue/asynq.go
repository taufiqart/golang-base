package queue

import (
	"context"

	"github.com/hibiken/asynq"
)

type AsynqClient struct {
	client *asynq.Client
}

func NewAsynqClient(redisAddr string, redisPassword string) *AsynqClient {
	client := asynq.NewClient(asynq.RedisClientOpt{
		Addr:     redisAddr,
		Password: redisPassword,
	})

	return &AsynqClient{
		client: client,
	}
}

func (c *AsynqClient) Enqueue(ctx context.Context, task *Task, opts ...EnqueueOptions) error {
	asynqTask := asynq.NewTask(task.Type, task.Payload)

	var asynqOpts []asynq.Option

	if len(opts) > 0 {
		opt := opts[0]
		if opt.ProcessIn > 0 {
			asynqOpts = append(asynqOpts, asynq.ProcessIn(opt.ProcessIn))
		}
		if opt.Queue != "" {
			asynqOpts = append(asynqOpts, asynq.Queue(opt.Queue))
		}
		if opt.MaxRetry > 0 {
			asynqOpts = append(asynqOpts, asynq.MaxRetry(opt.MaxRetry))
		}
	}

	_, err := c.client.EnqueueContext(ctx, asynqTask, asynqOpts...)
	return err
}

func (c *AsynqClient) Close() error {
	return c.client.Close()
}
