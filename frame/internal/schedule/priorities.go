package schedule

import "time"

const (
	PriorityLow Priority = iota
	PriorityNormal
	PriorityHigh
)

type (
	Priority uint8
)

const (
	runPriorityNotNeed  = -1
	runPriorityCritical = 2
)

var priorityAsMultiplier = map[Priority]float32{
	PriorityLow:    0.75,
	PriorityNormal: 1.00,
	PriorityHigh:   1.25,
}

type (
	Prioritize struct {
		// should return current time (time.Now())
		// redeclared for unit tests
		getTime timeObtainer
	}

	timeObtainer = func() time.Time
)

func NewPrioritize(obtainer timeObtainer) *Prioritize {
	return &Prioritize{
		getTime: obtainer,
	}
}

// should return value: -1, [0 to 100], +2
// where:
//  -1 - task excluded from running at all
//   0 - the lowest priority
//   1 - the highest priority
//   2 - task overdue, should be executed right now, without capacity check
func (p *Prioritize) calculateTaskPriority(task *Task) float32 {
	sinceLast := p.getTime().Sub(task.lastRunAt)

	if sinceLast < task.runAtMostOnceIn {
		// reject task that runs too often
		return runPriorityNotNeed
	}

	if sinceLast >= task.runAtLeastOnceIn {
		// overdue task
		return runPriorityCritical
	}

	// atLeast    = 1s
	// lastRun    = 9.1s
	// current    = 10s (after 900ms)
	// overdue    = 10.1s (left 100ms)
	// currentPos = 90/(60+120) = 0.5

	// (10s-9.1s)/(10.1s-9.1s) = 0.9%

	// whereIS:
	//        75%  | 100% | 150%
	// p    | low  | med  | hig
	// 0.00 | 0.00 | 0.00 | 0.00
	// 0.25 | 0.17 | 0.25 | 0.37
	// 0.50 | 0.35 | 0.50 | 0.75
	// 0.75 | 0.52 | 0.75 | 1.00
	// 1.00 | 1.00 | 1.00 | 1.00

	maxOverdueAt := task.lastRunAt.Add(task.runAtLeastOnceIn)
	currentPos := float64(p.getTime().UnixMicro()-task.lastRunAt.UnixMicro()) / float64(maxOverdueAt.UnixMicro()-task.lastRunAt.UnixMicro())

	return float32(currentPos) * priorityAsMultiplier[task.priority]
}
