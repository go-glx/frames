package frame

type (
	ErrBehavior uint
)

const (
	ErrBehaviorExit ErrBehavior = iota
	ErrBehaviorLog
)
