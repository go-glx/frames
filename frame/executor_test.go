package frame

import (
	"context"
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const testExampleDuration = time.Second * 1
const testExampleTicksRate = 24
const testExampleLatencyTick = time.Millisecond * 25  // 4 frame per 100ms / 40 frames per second
const testExampleLatencyFrame = time.Millisecond * 10 // 10 frame per 100ms / 100 frames per second
const testExampleLatencyTask = time.Millisecond * 5

func testExampleTask1(latency time.Duration) *Task {
	return NewTask(
		func() {
			// some additional task
			// will be executed only when we have free time
			// in frame (CPU more powerful than target FPS)

			// but it will be executed anyway at least
			// X time in X interval
			time.Sleep(latency)
		},
		WithRunAtLeastOnceIn(time.Second),
		WithRunAtMostOnceIn(time.Millisecond*500),
		WithPriority(TaskPriorityHigh),
	)
}

func testExampleTaskGC() *Task {
	return NewTask(
		func() {
			// try to run every 100ms
			// but at least once in second guaranteed
			runtime.GC()
			runtime.Gosched()
		},
		WithRunAtLeastOnceIn(time.Second),
		WithRunAtMostOnceIn(time.Millisecond*100),
		WithPriority(TaskPriorityLow),
	)
}

func TestExecutor_Execute(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testExampleDuration)
	defer cancel()

	start := time.Now()

	executor := NewExecutor(
		WithTargetTPS(testExampleTicksRate),
		WithTask(testExampleTask1(testExampleLatencyTask)),
		WithTask(testExampleTaskGC()),
		WithStatsCollector(func(stats Stats) {
			since := time.Since(start)

			fmt.Printf("|  %04dms |  %03d  | %02d/%02d |   %02d  |     %02dms |     %02dms |     %02dms |     %02dms |     %02dms |\n",
				since.Milliseconds(),
				stats.CycleID,
				stats.CurrentTPS, stats.TargetTPS,
				stats.CurrentFPS,

				stats.Rate.Milliseconds(),
				stats.Tick.Duration.Milliseconds(),
				stats.Frame.Duration.Milliseconds(),
				stats.Tasks.Duration.Milliseconds(),
				stats.ThrottleTime.Milliseconds(),
			)
		}),
	)

	fmt.Println("| -- STATS --                     | -- Frame --                                          |")
	fmt.Println("| elapsed | frame |  TPS  |  FPS  | capacity |   update |    frame |    tasks | throttle |")

	err := executor.Execute(ctx, func(_ TickStats) error {
		time.Sleep(testExampleLatencyTick)
		return nil
	}, func() error {
		time.Sleep(testExampleLatencyFrame)
		return nil
	})

	assert.NoError(t, err)
}
