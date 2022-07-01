package frame

import (
	"context"
	"time"

	"github.com/fe3dback/glx-frames/frame/internal/schedule"
)

const defaultLimitFPS = 60

type (
	Executor struct {
		tasks            []*Task
		logger           logger
		frameErrBehavior ErrBehavior
		limitFPS         int
		limitDuration    time.Duration

		// state
		scheduler   *schedule.Scheduler
		interrupted bool
	}

	mainFn        = func() error
	frameFinishFn = func(stats Stats)
)

func NewExecutor(initializers ...ExecutorInitializer) *Executor {
	e := &Executor{
		tasks:            []*Task{},
		logger:           &fallbackLogger{},
		frameErrBehavior: ErrBehaviorExit,
		limitFPS:         defaultLimitFPS,
		limitDuration:    time.Second / time.Duration(defaultLimitFPS),
	}

	for _, init := range initializers {
		init(e)
	}

	e.scheduler = schedule.NewScheduler(
		schedule.NewPrioritize(func() time.Time {
			return time.Now()
		}),
		transformTasks(e.tasks)...,
	)

	return e
}

func (e *Executor) Execute(ctx context.Context, fn mainFn, frameFinish frameFinishFn) error {
	e.listenForInterrupt(ctx)

	stats := Stats{
		CurrentFrame:   0,
		CurrentFPS:     e.limitFPS,
		FrameTargetFPS: e.limitFPS,
		FrameTimeLimit: e.limitDuration,
		Execute: Timings{
			StartAt: time.Now(),
		},
	}

	collectFPSAt := time.Time{}
	fps := 0

	for !e.interrupted {
		// prepare
		stats.CurrentFrame++
		stats.Frame.StartAt = time.Now()

		// run process
		stats.Process.StartAt = time.Now()
		err := fn()
		stats.Process.Duration = time.Since(stats.Process.StartAt)

		if err != nil {
			if next := e.handleError(err); next != nil {
				return next
			}
		}

		// calculate timings
		stats.FrameFreeTime = stats.FrameTimeLimit - stats.Process.Duration

		// run additional tasks, if we have free time
		stats.Tasks.StartAt = time.Now()
		if stats.FrameFreeTime > (time.Millisecond) {
			e.scheduler.Execute(stats.FrameFreeTime)
		}
		stats.Tasks.Duration = time.Since(stats.Tasks.StartAt)

		// throttle
		totalSpend := stats.Process.Duration + stats.Tasks.Duration
		stats.ThrottleTime = stats.FrameTimeLimit - totalSpend
		stats.FramePossibleFPS = int(time.Second / totalSpend)
		time.Sleep(stats.ThrottleTime)

		// end frame
		stats.Frame.Duration = time.Since(stats.Frame.StartAt)
		stats.Execute.Duration = time.Since(stats.Execute.StartAt)

		// calculate deltas
		stats.DeltaTime = stats.Frame.Duration.Seconds()

		// finish frame
		frameFinish(stats)

		// utils after frame processing
		fps++

		if collectFPSAt.After(time.Now()) {
			collectFPSAt = time.Now().Add(time.Second)
			stats.CurrentFPS = fps
			fps = 0
		}
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
	if e.frameErrBehavior == ErrBehaviorExit {
		return err
	}

	if e.frameErrBehavior == ErrBehaviorLog {
		e.logger.Error(err)
		return nil
	}

	return nil
}
