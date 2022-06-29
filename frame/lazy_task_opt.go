package frame

import "time"

type (
	LazyTaskInitializer = func(*LazyTask)
)

func WithRunAtLeastOnceIn(t time.Duration) LazyTaskInitializer {
	return func(lazyTask *LazyTask) {
		lazyTask.runAtLeastOnceIn = t
	}
}

func WithRunAtMostOnceIn(t time.Duration) LazyTaskInitializer {
	return func(lazyTask *LazyTask) {
		lazyTask.runAtMostOnceIn = t
	}
}

func WithPriority(p LazyTaskPriority) LazyTaskInitializer {
	return func(lazyTask *LazyTask) {
		lazyTask.priority = p
	}
}
