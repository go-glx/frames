package frame

import "time"

type Timings struct {
	StartAt  time.Time
	Duration time.Duration
}

type Stats struct {
	// DeltaTime is current frame duration in seconds.
	//
	// This useful for integrate into game logic update step
	// for example if you want to move Player at 100 units per second, you can do something like that:
	// 	player.x += 100*DeltaTime
	//  - If we have 60fps, DeltaTime=1s/60 = 0.0166666s (16ms)
	//  - If we have 30fps, DeltaTime=1s/30 = 0.0333333s (33ms)
	//
	// So:
	//  - (100*0.0166666) * 60 = 100
	//  - (100*0.0333333) * 30 = 100
	//
	// Typically game have unstable FPS, but player anyway will be moved at 100px per second, not depend on current rate
	// When cpu power > needed, this always will be "1s / FrameTargetFPS", because of throttling (see ThrottleTime)
	DeltaTime float64

	CurrentFrame uint64 // global frame ID since game start, every frame will increment this number on +1
	CurrentTPS   int    // real counted ticks per second (ticks is fixed/physics update)
	CurrentFPS   int    // real counted frames per second

	FrameFreeTime    time.Duration // how much free time we had in current frame (most likely was spent to tasks, GC, etc..)
	FrameTargetTPS   int           // target ticks per second (or fixed/physics update). Default is 50tps, but can be configured to any number
	FrameTargetFPS   int           // target frames per second. Default is 60fps, can be configured to any number
	FramePossibleFPS int           // maximum calculated FPS that can be theoretically achieved in current CPU
	FrameTimeLimit   time.Duration // how much time we have for processing in every frame. "= 1s / FrameTargetFPS"
	ThrottleTime     time.Duration // when CPU is more powerful when we need for processing at FrameTargetFPS rate, frame will sleep ThrottleTime in end of current frame

	Execute Timings // how long game is running
	Frame   Timings // current frame stats (process + fixes + tasks)
	Process Timings // timings for main update function
	Fixed   Timings // timings for fixed update function
	Tasks   Timings // timings for additional tasks that was executed in this frame
}
