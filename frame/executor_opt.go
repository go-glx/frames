package frame

import "time"

type (
	ExecutorInitializer = func(*Executor)
)

func WithFrameErrorHandleBehavior(behavior ErrBehavior) ExecutorInitializer {
	return func(e *Executor) {
		e.frameErrBehavior = behavior
	}
}

func WithTargetFPS(targetFPS int) ExecutorInitializer {
	return func(e *Executor) {
		e.limitFPS = targetFPS
		e.limitDuration = time.Second / time.Duration(targetFPS)
	}
}

func WithLogger(logger logger) ExecutorInitializer {
	return func(e *Executor) {
		e.logger = logger
	}
}
