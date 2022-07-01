package schedule

import (
	"github.com/stretchr/testify/assert"

	"testing"
	"time"
)

func testCreateTask(lastRun time.Time, mods ...func(*Task)) *Task {
	task := &Task{
		priority:         PriorityNormal,
		runAtLeastOnceIn: time.Second * 10,
		runAtMostOnceIn:  time.Millisecond * 100,
		lastRunAt:        lastRun,
		avgDuration:      time.Millisecond * 10,
		runsCount:        10,
	}

	for _, mod := range mods {
		mod(task)
	}

	return task
}

func Test_scheduler_Execute(t *testing.T) {
	const taskApple = "apple"
	const taskBanana = "banana"
	const taskOrange = "orange"

	currentTime := testMakeTime(30, 0)
	getTime := func() time.Time {
		return currentTime
	}

	executed50msAgo := currentTime.Add(-(time.Millisecond * 50))
	executed500msAgo := currentTime.Add(-(time.Millisecond * 500))
	executed1sAgo := currentTime.Add(-(time.Second))
	executed10sAgo := currentTime.Add(-(time.Second * 10))

	avgTime10ms := time.Millisecond * 10
	avgTime15ms := time.Millisecond * 15

	tests := []struct {
		name     string
		tasks    map[string]*Task
		capacity time.Duration
		expected []string
	}{
		{
			name: "by priority 2/3",
			tasks: map[string]*Task{
				// each task has 10ms duration
				taskBanana: testCreateTask(executed1sAgo),
				taskOrange: testCreateTask(executed1sAgo, func(task *Task) {
					task.priority = PriorityHigh
				}),
				taskApple: testCreateTask(executed500msAgo),
			},
			capacity: time.Millisecond * 21,
			expected: []string{
				taskOrange, // high priority
				taskBanana, // older than apple

				// apple - will not run, because 21ms capacity cover only 2 tasks with 10ms time
			},
		},
		{
			name: "long overdue 1/3",
			tasks: map[string]*Task{
				// each task has 10ms duration
				taskBanana: testCreateTask(executed10sAgo, func(task *Task) {
					// low, but overdue task
					task.priority = PriorityLow
					task.avgDuration = avgTime15ms
				}),
				taskOrange: testCreateTask(executed1sAgo, func(task *Task) {
					// high and old, second priority
					task.priority = PriorityHigh
					task.avgDuration = avgTime10ms
				}),
				taskApple: testCreateTask(executed500msAgo, func(task *Task) {
					task.avgDuration = avgTime10ms
				}),
			},
			capacity: time.Millisecond * 22,
			expected: []string{
				taskBanana, // 15ms / 22ms

				// other tasks need at least 10ms,
				// but we have only 7ms free
			},
		},
		{
			name: "two low priority, but fast runs 2/3",
			tasks: map[string]*Task{
				taskApple: testCreateTask(executed1sAgo, func(task *Task) {
					// runs first, high priority, long ago
					task.avgDuration = avgTime10ms
					task.priority = PriorityHigh
				}),
				taskBanana: testCreateTask(executed500msAgo, func(task *Task) {
					// runs second, high priority, but we not have capacity time to it
					// expected to have only 12ms left before run this
					// and its requirements is 15ms
					task.avgDuration = avgTime15ms
					task.priority = PriorityHigh
				}),
				taskOrange: testCreateTask(executed500msAgo, func(task *Task) {
					// low priority, but we have time to it
					task.avgDuration = avgTime10ms
					task.priority = PriorityLow
				}),
			},
			capacity: time.Millisecond * 21,
			expected: []string{
				taskApple,  // high
				taskOrange, // low, but only this possible to run
			},
		},
		{
			name: "nothing to run, because too often",
			tasks: map[string]*Task{
				taskApple: testCreateTask(executed50msAgo),
				taskBanana: testCreateTask(executed50msAgo, func(task *Task) {
					task.priority = PriorityHigh
				}),
				taskOrange: testCreateTask(executed50msAgo, func(task *Task) {
					task.priority = PriorityLow
				}),
			},
			capacity: time.Millisecond * 100, // has capacity for all!
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualResults := make([]string, 0)

			s := &Scheduler{
				prioritize: NewPrioritize(getTime),
				tasks:      testPrepareTasksToRun(tt.tasks, &actualResults),
			}
			s.Execute(tt.capacity)

			assert.Equal(t, tt.expected, actualResults, "executed tasks not match")
		})
	}
}

func testPrepareTasksToRun(tasks map[string]*Task, resultBuffer *[]string) []*Task {
	prepared := make([]*Task, 0, len(tasks))

	for name, task := range tasks {
		name, task := name, task
		task.taskFn = func() {
			time.Sleep(task.avgDuration)
			*resultBuffer = append(*resultBuffer, name)
		}

		prepared = append(prepared, task)
	}

	return prepared
}
