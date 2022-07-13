package frame

import (
	"time"
)

type (
	ExecutorInitializer = func(*Executor)
)

func WithTask(task *Task) ExecutorInitializer {
	return func(e *Executor) {
		e.tasks = append(e.tasks, task)
	}
}

func WithFrameErrorHandleBehavior(behavior ErrBehavior) ExecutorInitializer {
	return func(e *Executor) {
		e.frameErrBehavior = behavior
	}
}

func WithTargetFPS(targetFPS int) ExecutorInitializer {
	return func(e *Executor) {
		e.framePS = targetFPS
		e.frameDuration = time.Second / time.Duration(targetFPS)
	}
}

func WithTargetTPS(targetTPS int) ExecutorInitializer {
	return func(e *Executor) {
		e.ratePS = targetTPS
		e.rateDuration = time.Second / time.Duration(targetTPS)
	}
}

func WithLogger(logger logger) ExecutorInitializer {
	return func(e *Executor) {
		e.logger = logger
	}
}
