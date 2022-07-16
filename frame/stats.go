package frame

import "time"

type Timings struct {
	Start    time.Time
	Duration time.Duration
}

type TickStats struct {
	// CycleID is number of game loop cycles since game start (this will auto inc to +1 every loop)
	CycleID uint64

	// DeltaTime is physics/game update integration multiplier
	// for example if you want to move Player at 100 units per second, you can do something like that:
	// 	player.x += 100*DeltaTime
	//  - If we have 60tps, DeltaTime=1s/60 = 0.0166666s (16ms)
	//  - If we have 30tps, DeltaTime=1s/30 = 0.0333333s (33ms)
	//
	// So:
	//  - (100*0.0166666) * 60 = 100
	//  - (100*0.0333333) * 30 = 100
	//
	// This useful for integrate into game logic update step
	// Typically game have unstable FPS, but player anyway will be moved at 100px per second, not depend on current rate
	// When cpu power > needed, this always will be "1s / TargetTPS", because of throttling
	DeltaTime float64
}

type Stats struct {
	// CycleID is number of game loop cycles since game start (this will auto inc to +1 every loop)
	CycleID uint64

	// target ticks per second (state fixed/physics update per second).
	// This is how many state updates game will have per second
	TargetTPS int

	// maximum calculated FPS that can be theoretically achieved in current CPU
	PossibleFPS int

	// Rate is always 1s / TargetTPS
	Rate time.Duration

	Game         Timings       // game loop timings
	Cycle        Timings       // current cycle timings
	Tick         Timings       // current tick timings
	Frame        Timings       // current frame timings
	Tasks        Timings       // current cycle running tasks timings
	ThrottleTime time.Duration // when CPU is more powerful when we need for processing at TargetTPS rate, frame will sleep ThrottleTime in end of current cycle

	CurrentTPS int // real counted ticks per second (ticks is fixed/physics update)
	CurrentFPS int // real counted frames per second
}
