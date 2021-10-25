package runner

import (
	"context"
	"errors"
	"time"

	"github.com/euiko/tooyoul/mineman/lib/logger"
)

const Stop = time.Duration(-1)

var ErrGiveUp = errors.New("give up running an operation")

type (
	Operation interface {
		Run(ctx context.Context) error
	}

	OperationFunc func(ctx context.Context) error

	RetryStrategy interface {
		Next() time.Duration
		Reset()
	}

	NoRetry struct{}
)

func (o OperationFunc) Run(ctx context.Context) error {
	return o(ctx)
}

func (s *NoRetry) Next() time.Duration {
	return Stop
}

func (s *NoRetry) Reset() {}

func Run(ctx context.Context, operation Operation) SignalNotifier {
	return RunWithStrategy(ctx, operation, &NoRetry{})
}

func RunWithStrategy(ctx context.Context, operation Operation, strategy RetryStrategy) SignalNotifier {
	go run(ctx, operation, strategy)
	return &signalInterceptor{}
}

func run(ctx context.Context, operation Operation, strategy RetryStrategy) {
	newCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	for {
		err := doOperation(newCtx, operation)
		if err == nil {
			// reset retry strategy to mark that operation executed successfully
			strategy.Reset()
		}

		// it should be stoped when the program is give up
		if err != nil && err == ErrGiveUp {
			cancel()
		}

		// obtain next run
		next := strategy.Next()

		// cancel when stopped
		if next == Stop {
			cancel()
		}

		select {
		case <-newCtx.Done():
			return
		case <-time.After(next):
			break
		}

	}
}

func doOperation(ctx context.Context, operation Operation) error {
	defer func() {
		if err := recover(); err != nil {
			logger.Error(ctx, logger.MessageFmt("panic recovered in doOperation with error=%s", err))
		}
	}()

	return operation.Run(ctx)
}
