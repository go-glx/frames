package frame

import (
	"runtime"
	"time"
)

func NewDefaultTaskGarbageCollect() *Task {
	return NewTask(
		func() {
			runtime.GC()
			runtime.Gosched()
		},
		WithPriority(TaskPriorityLow),
		WithRunAtLeastOnceIn(time.Second*5),
		WithRunAtMostOnceIn(time.Millisecond*100),
	)
}
