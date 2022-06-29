package frame

import (
	"context"
	"fmt"
	"time"
)

const defaultLimitFPS = 60

type (
	Executor struct {
		logger           logger
		frameErrBehavior ErrBehavior
		limitFPS         int
		limitDuration    time.Duration

		// state
		currentFrameID  uint64
		executorStartAt time.Time
		frameStartAt    time.Time
		frameEndAt      time.Time
		frameDuration   time.Duration
		frameTimeLeft   time.Duration

		// system
		interrupted bool
	}

	mainFn = func() error
)

func NewExecutor(initializers ...ExecutorInitializer) *Executor {
	e := &Executor{
		logger:           &fallbackLogger{},
		frameErrBehavior: ErrBehaviorExit,
		limitFPS:         defaultLimitFPS,
		limitDuration:    time.Second / time.Duration(defaultLimitFPS),
	}

	for _, init := range initializers {
		init(e)
	}

	return e
}

func (e *Executor) Execute(ctx context.Context, fn mainFn) error {
	e.listenForInterrupt(ctx)
	e.executorStartAt = time.Now()
	e.currentFrameID = 0

	for !e.interrupted {
		e.currentFrameID++

		e.frameStartAt = time.Now()
		err := fn()
		e.frameEndAt = time.Now()

		if err != nil {
			if next := e.handleError(err); next != nil {
				return next
			}
		}

		e.frameDuration = e.frameEndAt.Sub(e.frameStartAt)
		e.frameTimeLeft = e.limitDuration - e.frameDuration

		// todo: run lazy tasks
		// todo: run before/after events
		// todo: calculate deltaTime
		// todo: continue next
	}

	return nil
}

func (e *Executor) listenForInterrupt(ctx context.Context) {
	e.interrupted = false

	go func() {
		<-ctx.Done()
		e.interrupted = true
	}()
}

func (e *Executor) handleError(err error) error {
	err = fmt.Errorf("error on %d frame: %w", e.currentFrameID, err)

	if e.frameErrBehavior == ErrBehaviorExit {
		return err
	}

	if e.frameErrBehavior == ErrBehaviorLog {
		e.logger.Error(err)
		return nil
	}

	return nil
}
