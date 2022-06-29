package frame

import "time"

type (
	timeRange struct {
		from time.Time
		to   time.Time
	}

	timeStats struct {
		elapsed time.Duration
		free    time.Duration
	}
)

func (tr *timeRange) start() {
	tr.from = time.Now()
}

func (tr *timeRange) finish() {
	tr.to = time.Now()
}
