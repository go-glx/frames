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
