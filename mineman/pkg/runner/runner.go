package runner

import (
	"context"
	"errors"
	"time"

	"github.com/euiko/tooyoul/mineman/pkg/log"
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
	return &signalInterceptor{
		handlers: []SignalHandler{},
	}
}

func run(ctx context.Context, operation Operation, strategy RetryStrategy) {

	newCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	defer func() {
		if err := recover(); err != nil {
			log.Error("panic recovered in runner.run", log.WithField("err", err))
			cancel()
		}
	}()

	for {
		// doOperation expect a blocking calls
		err := operation.Run(newCtx)
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
		}

	}
}
