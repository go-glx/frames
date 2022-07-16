package frame

import (
	"context"
	"math"
	"time"

	"github.com/go-glx/frames/frame/internal/schedule"
)

const defaultTPS = 60

type (
	Executor struct {
		tasks            []*Task
		logger           logger
		frameErrBehavior ErrBehavior
		targetTPS        int
		statsCollector   fnCollect

		// state
		interrupted bool
		scheduler   *schedule.Scheduler
		stats       Stats
	}

	fnCollect = func(stats Stats)
	fnTick    = func(tickStats TickStats) error
	fnDraw    = func() error
)

func NewExecutor(initializers ...ExecutorInitializer) *Executor {
	e := &Executor{
		tasks:            []*Task{},
		logger:           &fallbackLogger{},
		frameErrBehavior: ErrBehaviorExit,
		targetTPS:        defaultTPS,
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

func (e *Executor) Execute(ctx context.Context, updateFn fnTick, drawFn fnDraw) error {
	// handle cancel
	go func() {
		<-ctx.Done()
		e.interrupted = true
	}()

	// initialize loop state
	e.stats.CycleID = 0
	e.stats.TargetTPS = e.targetTPS
	e.stats.Rate = time.Second / time.Duration(e.stats.TargetTPS)
	e.stats.Game.Start = time.Now()
	e.stats.CurrentTPS = e.stats.TargetTPS
	e.stats.CurrentFPS = e.stats.TargetTPS

	// private state
	lastSyncAt := time.Now().Add(-e.stats.Rate)
	throttleCorrection := time.Duration(0)
	resetCountersAt := time.Now().Add(time.Second)
	currentTPS := 0
	currentFPS := 0

	for !e.interrupted {
		// Start
		// -------------------------
		e.stats.CycleID++
		e.stats.Cycle.Start = time.Now()

		deltaTime := e.stats.Cycle.Start.Sub(lastSyncAt)
		lastSyncAt = lastSyncAt.Add(deltaTime)

		// calculate throttle correction
		// this will snap loop cycles to Rate intervals
		idealStartAt := e.stats.Game.Start.Add(
			time.Duration(e.stats.CycleID-1) * e.stats.Rate,
		)

		diffFromIdeal := e.stats.Cycle.Start.Sub(idealStartAt).Microseconds()
		diffFromIdeal = int64(math.Mod(float64(diffFromIdeal), float64(e.stats.Rate.Microseconds())))
		throttleCorrection = time.Duration(diffFromIdeal) * time.Microsecond

		// Tick
		// -------------------------
		e.stats.Tick.Start = time.Now()
		updateDelta := e.stats.Rate + deltaTime
		requiredUpdate := true

		for updateDelta > e.stats.Rate {
			if requiredUpdate {
				// this will guarantee one update call every cycle
				// if deltaTime less that Rate,
				// but we throttle each frame to all not used budget
				// anyway, so not needed updates will not run
				requiredUpdate = false
				updateDelta -= e.stats.Rate
			}

			currentTPS++
			err := updateFn(TickStats{
				CycleID:   e.stats.CycleID,
				DeltaTime: deltaTime.Seconds(),
			})
			if err != nil {
				if nextErr := e.handleError(err); nextErr != nil {
					return nextErr
				}
			}

			updateDelta -= e.stats.Rate
		}
		e.stats.Tick.Duration = time.Since(e.stats.Tick.Start)

		// Frame
		// -------------------------
		e.stats.Frame.Start = time.Now()
		currentFPS++
		err := drawFn()
		if err != nil {
			if nextErr := e.handleError(err); nextErr != nil {
				return nextErr
			}
		}
		e.stats.Frame.Duration = time.Since(e.stats.Frame.Start)

		// Tasks
		// -------------------------
		totalSpend := e.stats.Tick.Duration + e.stats.Frame.Duration
		freeTime := e.stats.Rate - totalSpend
		e.stats.PossibleFPS = int(time.Second / totalSpend)

		e.stats.Tasks.Start = time.Now()
		e.scheduler.Execute(freeTime)
		e.stats.Tasks.Duration = time.Since(e.stats.Tasks.Start)

		// Throttle
		// -------------------------
		timeTaken := 0 +
			e.stats.Tick.Duration +
			e.stats.Frame.Duration +
			e.stats.Tasks.Duration

		e.stats.ThrottleTime = e.stats.Rate - timeTaken

		if throttleCorrection > 0 {
			e.stats.ThrottleTime -= throttleCorrection
		}

		if e.stats.ThrottleTime < 0 {
			e.stats.ThrottleTime = 0
		}

		time.Sleep(e.stats.ThrottleTime)

		// End
		// -------------------------
		e.stats.Cycle.Duration = time.Since(e.stats.Cycle.Start)
		e.stats.Game.Duration = time.Since(e.stats.Game.Start)

		if time.Now().After(resetCountersAt) {
			resetCountersAt = time.Now().Add(time.Second)
			e.stats.CurrentTPS = currentTPS
			e.stats.CurrentFPS = currentFPS
			currentTPS = 0
			currentFPS = 0
		}

		if e.statsCollector != nil {
			e.statsCollector(e.stats)
		}
	}

	return nil
}

// func (e *Executor) frameUpdate(mainUpdate fnTick, frameFinish frameFinishFn, errChannel chan<- error) {
// 	lastSyncAt := time.Now()
//
//
// 	for !e.interrupted {
// 		// ------------------------------
// 		e.mux.Lock()
// 		// ------------------------------
// 		// prepare
// 		e.deprecatedStats.CurrentFrame++
// 		e.deprecatedStats.Frame.Start = time.Now()
// 		e.deprecatedStats.Fixed.Duration = 0
//

//
// 		// run process
// 		e.deprecatedStats.Process.Start = time.Now()
// 		err := mainUpdate()
// 		e.deprecatedStats.Process.Duration = time.Since(e.deprecatedStats.Process.Start)
//
// 		if err != nil {
// 			if next := e.handleError(err); next != nil {
// 				errChannel <- next
// 				break
// 			}
// 		}
//
// 		// calculate timings
// 		e.deprecatedStats.FrameFreeTime = e.deprecatedStats.FrameTimeLimit - e.deprecatedStats.Process.Duration - e.deprecatedStats.Fixed.Duration
//
// 		// run additional tasks, if we have free time
// 		e.deprecatedStats.Tasks.Start = time.Now()
// 		e.scheduler.Execute(e.deprecatedStats.FrameFreeTime)
// 		e.deprecatedStats.Tasks.Duration = time.Since(e.deprecatedStats.Tasks.Start)
//
// 		// calculate throttle
// 		totalSpend := e.deprecatedStats.Process.Duration + e.deprecatedStats.Fixed.Duration + e.deprecatedStats.Tasks.Duration
// 		e.deprecatedStats.ThrottleTime = e.deprecatedStats.FrameTimeLimit - totalSpend
// 		if throttleCorrection > 0 {
// 			e.deprecatedStats.ThrottleTime -= throttleCorrection
// 		}
//
//
//
// 		// end frame
// 		e.deprecatedStats.Frame.Duration = time.Since(e.deprecatedStats.Frame.Start)
// 		e.deprecatedStats.Execute.Duration = time.Since(e.deprecatedStats.Execute.Start)
//
// 		// calculate deltas
// 		e.deprecatedStats.DeltaTime = time.Since(lastSyncAt).Seconds()
// 		lastSyncAt = time.Now()
// 		e.realFPS++
//
// 		// finish frame
// 		frameFinish(e.deprecatedStats)
//
// 		// unlock game loop, next we just wait for next frame
// 		// and give time for other goroutines to work (fixed update, deprecatedStats)
// 		// ------------------------------
// 		e.mux.Unlock()
// 		// ------------------------------
//
// 		if e.deprecatedStats.ThrottleTime > 0 {
// 			time.Sleep(e.deprecatedStats.ThrottleTime)
// 		}
// 	}
// }
//
// func (e *Executor) fixedUpdate(fixedUpdate fnTick, errChannel chan<- error) {
// 	updateInterval := time.NewTicker(e.rateDuration)
// 	defer updateInterval.Stop()
//
// fixedUpdate:
// 	for !e.interrupted {
// 		select {
// 		case <-updateInterval.C:
// 			e.mux.Lock()
//
// 			e.deprecatedStats.Fixed.Start = time.Now()
// 			err := fixedUpdate()
// 			if err != nil {
// 				if next := e.handleError(err); next != nil {
// 					errChannel <- next
// 					break fixedUpdate
// 				}
// 			}
// 			e.realTPS++
// 			e.deprecatedStats.Fixed.Duration += time.Since(e.deprecatedStats.Fixed.Start)
//
// 			e.mux.Unlock()
// 		}
// 	}
// }
//
// func (e *Executor) calculatePerformance() {
// 	for !e.interrupted {
// 		select {
// 		case <-time.After(time.Second):
// 			e.mux.Lock()
//
// 			e.deprecatedStats.CurrentFPS = e.realFPS
// 			e.deprecatedStats.CurrentTPS = e.realTPS
// 			e.realFPS = 0
// 			e.realTPS = 0
//
// 			e.mux.Unlock()
// 		}
// 	}
// }

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
