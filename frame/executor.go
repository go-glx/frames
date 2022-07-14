package frame

import (
	"context"
	"sync"
	"time"

	"github.com/go-glx/frames/frame/internal/schedule"
)

const defaultLimitFPS = 60
const defaultTPS = 50

type (
	Executor struct {
		tasks            []*Task
		logger           logger
		frameErrBehavior ErrBehavior
		framePS          int           // target FPS
		frameDuration    time.Duration // 1s/FPS
		ratePS           int           // physic/static updates per seconds (TPS, or ticks per second)
		rateDuration     time.Duration // ticks interval

		// state
		lastSyncAt  time.Time
		realFPS     int
		realTPS     int
		stats       Stats
		scheduler   *schedule.Scheduler
		interrupted bool

		mux sync.Mutex
	}

	updateFn      = func() error
	frameFinishFn = func(stats Stats)
)

func NewExecutor(initializers ...ExecutorInitializer) *Executor {
	e := &Executor{
		tasks:            []*Task{},
		logger:           &fallbackLogger{},
		frameErrBehavior: ErrBehaviorExit,
		framePS:          defaultLimitFPS,
		frameDuration:    time.Second / time.Duration(defaultLimitFPS),
		ratePS:           defaultTPS,
		rateDuration:     time.Second / time.Duration(defaultTPS),
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

	e.stats = Stats{
		CurrentFrame:   0,
		CurrentTPS:     e.ratePS,
		FrameTargetTPS: e.ratePS,
		CurrentFPS:     e.framePS,
		FrameTargetFPS: e.framePS,
		FrameTimeLimit: e.frameDuration,
		Execute: Timings{
			StartAt: time.Now(),
		},
	}

	return e
}

func (e *Executor) Execute(ctx context.Context, mainUpdate updateFn, fixedUpdate updateFn, frameFinish frameFinishFn) error {
	errChan := make(chan error, 1)

	go e.calculatePerformance()
	go e.frameUpdate(mainUpdate, frameFinish, errChan)

	if e.ratePS > 0 {
		go e.fixedUpdate(fixedUpdate, errChan)
	}

	select {
	case err := <-errChan:
		e.interrupted = true
		return err
	case <-ctx.Done():
		e.interrupted = true
		return nil
	}
}

func (e *Executor) frameUpdate(mainUpdate updateFn, frameFinish frameFinishFn, errChannel chan<- error) {
	e.lastSyncAt = time.Now()

	for !e.interrupted {
		// ------------------------------
		e.mux.Lock()
		// ------------------------------
		// prepare
		e.stats.CurrentFrame++
		e.stats.Frame.StartAt = time.Now()
		e.stats.Fixed.Duration = 0

		// run process
		e.stats.Process.StartAt = time.Now()
		err := mainUpdate()
		e.stats.Process.Duration = time.Since(e.stats.Process.StartAt)

		if err != nil {
			if next := e.handleError(err); next != nil {
				errChannel <- next
				break
			}
		}

		// calculate timings
		e.stats.FrameFreeTime = e.stats.FrameTimeLimit - e.stats.Process.Duration - e.stats.Fixed.Duration

		// run additional tasks, if we have free time
		e.stats.Tasks.StartAt = time.Now()
		e.scheduler.Execute(e.stats.FrameFreeTime)
		e.stats.Tasks.Duration = time.Since(e.stats.Tasks.StartAt)

		// calculate throttle
		totalSpend := e.stats.Process.Duration + e.stats.Fixed.Duration + e.stats.Tasks.Duration
		e.stats.ThrottleTime = e.stats.FrameTimeLimit - totalSpend
		e.stats.FramePossibleFPS = int(time.Second / totalSpend)

		// end frame
		e.stats.Frame.Duration = time.Since(e.stats.Frame.StartAt)
		e.stats.Execute.Duration = time.Since(e.stats.Execute.StartAt)

		// calculate deltas
		e.stats.DeltaTime = time.Since(e.lastSyncAt).Seconds()
		e.lastSyncAt = time.Now()
		e.realFPS++

		// finish frame
		frameFinish(e.stats)

		// unlock game loop, next we just wait for next frame
		// and give time for other goroutines to work (fixed update, stats)
		// ------------------------------
		e.mux.Unlock()
		// ------------------------------

		if e.stats.ThrottleTime > 0 {
			time.Sleep(e.stats.ThrottleTime)
		}

		// todo:

		// throttleStart := time.Now()
		// sleepTimeLeft := expectedThrottleTime
		//
		// // mini throttle sleep loop, we wait to spend all free time
		// // and give chance to execute other threads (fixed update)
		// for sleepTimeLeft > 0 {
		// 	e.mux.Lock()
		// 	// correct free time by fixed update from last check
		// 	if fixedUpdateSpent != e.stats.Fixed.Duration {
		// 		fixedUpdateSpent = e.stats.Fixed.Duration
		// 		sleepTimeLeft -= e.stats.Fixed.Duration
		// 	}
		// 	e.mux.Unlock()
		//
		// 	// if fixed update eat all free time, just exit
		// 	if sleepTimeLeft <= 0 {
		// 		break
		// 	}
		//
		// 	// if free time <1ms, sleep for remaining and exit
		// 	if sleepTimeLeft < time.Millisecond {
		// 		time.Sleep(sleepTimeLeft)
		// 		break
		// 	}
		//
		// 	// ok, wait for next sync in 1ms
		// 	time.Sleep(time.Millisecond)
		// 	sleepTimeLeft -= time.Millisecond
		// }
		//
		// e.stats.ThrottleTime = time.Since(throttleStart)
		// penaltyTime = e.stats.ThrottleTime - expectedThrottleTime
		// if penaltyTime < 0 {
		// 	penaltyTime = 0
		// }
	}
}

func (e *Executor) fixedUpdate(fixedUpdate updateFn, errChannel chan<- error) {
	updateInterval := time.NewTicker(e.rateDuration)
	defer updateInterval.Stop()

fixedUpdate:
	for !e.interrupted {
		select {
		case <-updateInterval.C:
			e.mux.Lock()

			e.stats.Fixed.StartAt = time.Now()
			err := fixedUpdate()
			if err != nil {
				if next := e.handleError(err); next != nil {
					errChannel <- next
					break fixedUpdate
				}
			}
			e.realTPS++
			e.stats.Fixed.Duration += time.Since(e.stats.Fixed.StartAt)

			e.mux.Unlock()
		}
	}
}

func (e *Executor) calculatePerformance() {
	for !e.interrupted {
		select {
		case <-time.After(time.Second):
			e.mux.Lock()

			e.stats.CurrentFPS = e.realFPS
			e.stats.CurrentTPS = e.realTPS
			e.realFPS = 0
			e.realTPS = 0

			e.mux.Unlock()
		}
	}
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
