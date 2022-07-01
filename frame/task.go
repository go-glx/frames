package frame

import "time"

type TaskPriority uint

const (
	TaskPriorityLow TaskPriority = iota
	TaskPriorityNormal
	TaskPriorityHigh
)

type Task struct {
	fn               func()
	priority         TaskPriority  // task schedule priority against another tasks
	runAtLeastOnceIn time.Duration // but anyway it SHOULD be executed at least once per X time
	runAtMostOnceIn  time.Duration // do not run it too often
}

func NewTask(fn func(), options ...TaskInitializer) *Task {
	task := &Task{
		fn:               fn,
		priority:         TaskPriorityNormal,
		runAtLeastOnceIn: time.Minute,
		runAtMostOnceIn:  time.Second,
	}

	for _, init := range options {
		init(task)
	}

	return task
}
