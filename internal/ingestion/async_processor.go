package ingestion

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
)

var ErrAsyncProcessorQueueFull = errors.New("async processor queue is full")

// AsyncProcessor processes telemetry readings in background worker goroutines.
type AsyncProcessor struct {
	logger      *slog.Logger
	service     *Service
	workerCount int
	queue       chan TelemetryInput
	wg          sync.WaitGroup
}

// NewAsyncProcessor creates a background telemetry processor.
func NewAsyncProcessor(
	logger *slog.Logger,
	service *Service,
	workerCount int,
	queueSize int,
) *AsyncProcessor {
	if workerCount <= 0 {
		workerCount = 1
	}

	if queueSize <= 0 {
		queueSize = 100
	}

	return &AsyncProcessor{
		logger:      logger,
		service:     service,
		workerCount: workerCount,
		queue:       make(chan TelemetryInput, queueSize),
	}
}

// Start launches worker goroutines.
func (p *AsyncProcessor) Start(ctx context.Context) {
	for workerID := 1; workerID <= p.workerCount; workerID++ {
		p.wg.Add(1)

		go func(workerID int) {
			defer p.wg.Done()
			p.runWorker(ctx, workerID)
		}(workerID)
	}
}

// Submit puts telemetry input into the processing queue.
func (p *AsyncProcessor) Submit(ctx context.Context, input TelemetryInput) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case p.queue <- input:
		return nil
	default:
		return ErrAsyncProcessorQueueFull
	}
}

// Wait waits until all workers stop.
// Workers stop when the context passed to Start is cancelled.
func (p *AsyncProcessor) Wait() {
	p.wg.Wait()
}

func (p *AsyncProcessor) runWorker(ctx context.Context, workerID int) {
	p.logger.Info("async telemetry worker started", "workerId", workerID)

	for {
		select {
		case <-ctx.Done():
			p.logger.Info("async telemetry worker stopped", "workerId", workerID)
			return

		case input := <-p.queue:
			if _, err := p.service.Process(ctx, input); err != nil {
				p.logger.Error(
					"async telemetry processing failed",
					"workerId", workerID,
					"parameterType", input.ParameterType,
					"sourceId", input.SourceID,
					"error", err,
				)

				continue
			}

			p.logger.Debug(
				"async telemetry processed",
				"workerId", workerID,
				"parameterType", input.ParameterType,
				"sourceId", input.SourceID,
			)
		}
	}
}

func (p *AsyncProcessor) QueueLength() int {
	return len(p.queue)
}

func (p *AsyncProcessor) String() string {
	return fmt.Sprintf("AsyncProcessor{workers=%d, queueLength=%d}", p.workerCount, len(p.queue))
}
