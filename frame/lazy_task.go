package frame

import (
	"sort"
	"time"
)

const (
	LazyTaskPriorityLow LazyTaskPriority = iota
	LazyTaskPriorityNormal
	LazyTaskPriorityHigh
)

const (
	taskRunPriorityNotNeed  = -1
	taskRunPriorityCritical = 2
)

const (
	priorityMultiplierLow    = 0.75
	priorityMultiplierNormal = 1
	priorityMultiplierHigh   = 1.25
)

type (
	LazyTaskPriority uint8

	LazyTask struct {
		priority         LazyTaskPriority // task schedule priority against another tasks
		runAtLeastOnceIn time.Duration    // but anyway it SHOULD be executed at least once per X time
		runAtMostOnceIn  time.Duration    // do not run it too often
		task             task

		// inner stats
		currentPriority    float32 // -1; [0..1]; +2
		priorityMultiplier float32
		lastRunAt          time.Time
		avgDuration        time.Duration // how much time this task taken in avg
	}

	task = func()

	scheduler struct {
		lazyTasks []*LazyTask
	}
)

func NewLazyTask(task task, options ...LazyTaskInitializer) *LazyTask {
	lt := &LazyTask{
		priority:         LazyTaskPriorityLow,
		runAtLeastOnceIn: time.Second * 30,
		runAtMostOnceIn:  time.Second * 5,
		task:             task,
	}

	for _, init := range options {
		init(lt)
	}

	// calculate system values
	lt.priorityMultiplier = priorityAsMultiplier(lt.priority)

	return lt
}

// should return value from -1 or [0 to 100]
// where:
//  -1 - task excluded from running at all
//   0 - the lowest priority
//   1 - the highest priority
//   2 - task overdue, should be executed right now, without capacity check
func calculateTaskPriority(task *LazyTask) float32 {
	sinceLast := time.Since(task.lastRunAt)

	if sinceLast < task.runAtMostOnceIn {
		// reject task that runs too often
		return taskRunPriorityNotNeed
	}

	if sinceLast > task.runAtLeastOnceIn {
		// overdue task
		return taskRunPriorityCritical
	}

	// lastRun    = 60
	// current    = 90
	// overdue    = 120
	// currentPos = 90/(60+120) = 0.5

	// whereIS:
	//        75%  | 100% | 150%
	// p    | low  | med  | hig
	// 0.00 | 0.00 | 0.00 | 0.00
	// 0.25 | 0.17 | 0.25 | 0.37
	// 0.50 | 0.35 | 0.50 | 0.75
	// 0.75 | 0.52 | 0.75 | 1.00
	// 1.00 | 1.00 | 1.00 | 1.00

	maxOverdueAt := task.lastRunAt.Add(task.runAtLeastOnceIn)
	currentPos := time.Now().UnixMicro() / (task.lastRunAt.UnixMicro() + maxOverdueAt.UnixMicro())

	return float32(currentPos) * task.priorityMultiplier
}

func priorityAsMultiplier(p LazyTaskPriority) float32 {
	if p == LazyTaskPriorityHigh {
		return priorityMultiplierHigh
	}

	if p == LazyTaskPriorityLow {
		return priorityMultiplierLow
	}

	return priorityMultiplierNormal
}

func newScheduler(tasks []*LazyTask) *scheduler {
	return &scheduler{
		lazyTasks: tasks,
	}
}

func (s *scheduler) Execute(freeTime time.Duration) {
	taskIDs := s.schedule(freeTime)
	if len(taskIDs) == 0 {
		return
	}

	for taskID := range taskIDs {
		s.lazyTasks[taskID].task()
	}
}

func (s *scheduler) schedule(capacity time.Duration) []int {
	taskIDs := make([]int, 0)

	for _, lazyTask := range s.lazyTasks {
		lazyTask.currentPriority = calculateTaskPriority(lazyTask)
	}

	sort.Slice(s.lazyTasks, func(i, j int) bool {
		return s.lazyTasks[i].currentPriority <= s.lazyTasks[j].currentPriority
	})

	for id, lazyTask := range s.lazyTasks {
		if lazyTask.currentPriority == taskRunPriorityNotNeed {
			// not need run this task right now
			continue
		}

		if lazyTask.currentPriority == taskRunPriorityCritical {
			// should be executed right now
			taskIDs = append(taskIDs, id)
			capacity = capacity - lazyTask.avgDuration
			continue
		}

		if capacity <= 0 {
			break
		}

		if lazyTask.avgDuration <= 0 {
			// don`t known duration yet, possible > capacity
			// so run only this at current frame
			taskIDs = append(taskIDs, id)
			break
		}

		if lazyTask.avgDuration < capacity {
			// not have time to it
			continue
		}
	}

	return taskIDs
}
