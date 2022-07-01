package frame

import "time"

type Timings struct {
	StartAt  time.Time
	Duration time.Duration
}

type Stats struct {
	CurrentFrame uint64
	CurrentFPS   int
	DeltaTime    float64

	FrameFreeTime    time.Duration
	FrameTargetFPS   int
	FramePossibleFPS int
	FrameTimeLimit   time.Duration
	ThrottleTime     time.Duration

	Execute Timings
	Frame   Timings
	Process Timings
	Tasks   Timings
}
