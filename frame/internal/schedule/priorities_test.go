package schedule

import (
	"testing"
	"time"
)

func testMakeTime(sec int, ms int) time.Time {
	return time.Date(2000, 01, 01, 12, 15, sec, ms*1000000, time.Local)
}

func Test_calculateTaskPriority(t *testing.T) {
	currentTime := testMakeTime(10, 0)

	tests := []struct {
		name        string
		currentTime time.Time
		task        *Task
		want        float32
	}{
		{
			name:        "critical (runAtLeastOnceIn overdue)",
			currentTime: currentTime,
			task: &Task{
				priority:         PriorityNormal,
				runAtLeastOnceIn: time.Second,
				runAtMostOnceIn:  time.Millisecond * 100,
				lastRunAt:        currentTime.Add(-(time.Second * 2)),
			},
			want: runPriorityCritical,
		},
		{
			name:        "critical but, skip, because runs too often",
			currentTime: currentTime,
			task: &Task{
				priority:         PriorityNormal,
				runAtLeastOnceIn: time.Second,
				runAtMostOnceIn:  time.Second * 10,
				lastRunAt:        currentTime.Add(-(time.Second * 3)),
			},
			want: runPriorityNotNeed,
		},
		{
			name:        "NORMAL (1.0): (last) 500ms .. now .. 500ms (atLeast)",
			currentTime: currentTime,
			task: &Task{
				priority:         PriorityNormal,
				runAtLeastOnceIn: time.Second,
				runAtMostOnceIn:  time.Millisecond * 100,
				lastRunAt:        currentTime.Add(-(time.Millisecond * 500)),
			},
			want: 0.5,
		},
		{
			name:        "LOW (0.75): (last) 500ms .. now .. 500ms (atLeast)",
			currentTime: currentTime,
			task: &Task{
				priority:         PriorityLow,
				runAtLeastOnceIn: time.Second,
				runAtMostOnceIn:  time.Millisecond * 100,
				lastRunAt:        currentTime.Add(-(time.Millisecond * 500)),
			},
			want: 0.375,
		},
		{
			name:        "HIGH (1.25): (last) 500ms .. now .. 500ms (atLeast)",
			currentTime: currentTime,
			task: &Task{
				priority:         PriorityHigh,
				runAtLeastOnceIn: time.Second,
				runAtMostOnceIn:  time.Millisecond * 100,
				lastRunAt:        currentTime.Add(-(time.Millisecond * 500)),
			},
			want: 0.625,
		},
		{
			name:        "NORMAL (90%): (last) 900ms .. now .. 100ms (atLeast)",
			currentTime: currentTime,
			task: &Task{
				priority:         PriorityNormal,
				runAtLeastOnceIn: time.Second,
				runAtMostOnceIn:  time.Millisecond * 100,
				lastRunAt:        currentTime.Add(-(time.Millisecond * 900)),
			},
			want: 0.9,
		},
		{
			name:        "HIGH (90%): (last) 900ms .. now .. 100ms (atLeast)",
			currentTime: currentTime,
			task: &Task{
				priority:         PriorityHigh,
				runAtLeastOnceIn: time.Second,
				runAtMostOnceIn:  time.Millisecond * 100,
				lastRunAt:        currentTime.Add(-(time.Millisecond * 900)),
			},
			want: 1.125,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewPrioritize(func() time.Time {
				return tt.currentTime
			})

			if got := service.calculateTaskPriority(tt.task); got != tt.want {
				t.Errorf("calculateTaskPriority() = %v, want %v", got, tt.want)
			}
		})
	}
}
