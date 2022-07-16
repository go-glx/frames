package frame

import (
	"fmt"
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

func WithStatsCollector(collector fnCollect) ExecutorInitializer {
	return func(e *Executor) {
		e.statsCollector = collector
	}
}

func WithTargetTPS(targetTPS int) ExecutorInitializer {
	return func(e *Executor) {
		if targetTPS <= 0 {
			panic(fmt.Errorf("TargetTPS should be greater than zero"))
		}

		e.targetTPS = targetTPS
	}
}

func WithLogger(logger logger) ExecutorInitializer {
	return func(e *Executor) {
		e.logger = logger
	}
}
