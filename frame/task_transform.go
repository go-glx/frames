package frame

import "github.com/fe3dback/glx-frames/frame/internal/schedule"

func transformTasks(tasks []*Task) []*schedule.Task {
	innerTasks := make([]*schedule.Task, 0, len(tasks))

	for _, task := range tasks {
		innerTasks = append(innerTasks, transformTaskToInternal(task))
	}

	return innerTasks
}

func transformTaskToInternal(task *Task) *schedule.Task {
	return schedule.NewTask(
		task.fn,
		transformTaskPriorityToInternal(task.priority),
		task.runAtLeastOnceIn,
		task.runAtMostOnceIn,
	)
}

func transformTaskPriorityToInternal(p TaskPriority) schedule.Priority {
	switch p {
	case TaskPriorityLow:
		return schedule.PriorityLow
	case TaskPriorityHigh:
		return schedule.PriorityHigh
	default:
		return schedule.PriorityNormal
	}
}
