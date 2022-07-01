package frame

import "time"

type (
	TaskInitializer = func(*Task)
)

func WithRunAtLeastOnceIn(t time.Duration) TaskInitializer {
	return func(task *Task) {
		task.runAtLeastOnceIn = t
	}
}

func WithRunAtMostOnceIn(t time.Duration) TaskInitializer {
	return func(task *Task) {
		task.runAtMostOnceIn = t
	}
}

func WithPriority(p TaskPriority) TaskInitializer {
	return func(task *Task) {
		task.priority = p
	}
}
