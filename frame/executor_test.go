package frame

import (
	"context"
	"fmt"
	"runtime"
	"testing"
	"time"
)

func TestExecutor_Execute(t *testing.T) {
	const testTime = time.Second * 1
	const ticksRate = 24
	const frameTimeAvg = time.Millisecond * 25 // 4 frame per 100ms / 40 frames per second
	const tickTimeAvg = time.Millisecond * 10  // 10 frame per 100ms / 100 frames per second

	ctx, cancel := context.WithTimeout(context.Background(), testTime)
	defer cancel()

	executor := NewExecutor(
		WithTargetTPS(ticksRate),
		WithTask(
			NewTask(
				func() {
					// some additional task
					// will be executed only when we have free time
					// in frame (CPU more powerful than target FPS)

					// but it will be executed anyway at least
					// X time in X interval
					time.Sleep(time.Millisecond * 5)
				},
				WithRunAtLeastOnceIn(time.Second),
				WithRunAtMostOnceIn(time.Millisecond*500),
				WithPriority(TaskPriorityHigh),
			),
		),
		WithTask(
			NewTask(
				func() {
					// try to run every 100ms
					// but at least once in second guaranteed
					runtime.GC()
					runtime.Gosched()
				},
				WithRunAtLeastOnceIn(time.Second),
				WithRunAtMostOnceIn(time.Millisecond*100),
				WithPriority(TaskPriorityLow),
			),
		),
	)

	// start := time.Now()
	_ = ctx      // todo remove
	_ = executor // todo remove

	fmt.Println("| -- STATS --                     | -- Frame --                               |")
	fmt.Println("| elapsed | frame |  FPS  |  TPS  | capacity |       fn |    fixed |    tasks | throttle |")

	// todo
	// err := executor.Execute(ctx, func() error {
	// 	time.Sleep(frameTimeAvg) // fn time
	// 	return nil
	// }, func() error {
	// 	time.Sleep(tickTimeAvg) // fixed time
	// 	return nil
	// }, func(stats DeprecatedStats) {
	// 	since := time.Since(start)
	//
	// 	fmt.Printf("|  %04dms |  %03d  | %02d/%02d | %02d/%02d |     %02dms |     %02dms |     %02dms |     %02dms |     %02dms |\n",
	// 		since.Milliseconds(),
	// 		stats.CurrentFrame,
	// 		stats.CurrentFPS, stats.FrameTargetFPS,
	// 		stats.CurrentTPS, stats.FrameTargetTPS,
	//
	// 		stats.FrameTimeLimit.Milliseconds(),
	// 		stats.Process.Duration.Milliseconds(),
	// 		stats.Fixed.Duration.Milliseconds(),
	// 		stats.Tasks.Duration.Milliseconds(),
	// 		stats.ThrottleTime.Milliseconds(),
	// 	)
	// })
	//
	// assert.NoError(t, err)
}
