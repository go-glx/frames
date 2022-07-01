package schedule

import (
	"sort"
	"time"
)

type Scheduler struct {
	prioritize *Prioritize
	tasks      []*Task
}

func NewScheduler(prioritize *Prioritize, tasks ...*Task) *Scheduler {
	return &Scheduler{
		prioritize: prioritize,
		tasks:      tasks,
	}
}

func (s *Scheduler) Execute(capacity time.Duration) {
	for _, lazyTask := range s.tasks {
		lazyTask.currentPriority = s.prioritize.calculateTaskPriority(lazyTask)
	}

	sort.Slice(s.tasks, func(i, j int) bool {
		return s.tasks[i].currentPriority >= s.tasks[j].currentPriority
	})

	for _, task := range s.tasks {
		if task.currentPriority == runPriorityNotNeed {
			// not need run this task right now
			continue
		}

		if task.currentPriority == runPriorityCritical {
			// should be executed right now
			capacity -= s.run(task)
			continue
		}

		if capacity <= 0 {
			break
		}

		if task.avgDuration <= 0 {
			// don`t known duration yet, possible > capacity
			// so run only this at current frame
			s.run(task)
			break
		}

		if task.avgDuration > capacity {
			// not have time to it
			continue
		}

		capacity -= s.run(task)
	}
}

// Run function and return it duration
func (s *Scheduler) run(task *Task) time.Duration {
	task.lastRunAt = time.Now()
	task.taskFn()
	duration := time.Since(task.lastRunAt)

	task.avgDuration = ((task.avgDuration * time.Duration(task.runsCount)) + duration) /
		(time.Duration(task.runsCount) + 1)

	task.runsCount++
	return duration
}
