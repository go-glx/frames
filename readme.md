# glx-frames

Library for making endless game-loops, it`s heart of any game engine.

## Base usage

Executor will run your `gameLoop` and automatic collect
and calculate all frame stats, also it will be throttle
processing, when CPU is more powerful than we need in `targetFPS`

After each frame, it call `afterFrame` function with all
frame stats, and most useful is frame `stats.DeltaTime`

```go

import "github.com/fe3dback/glx-frames/frame"

func main() {
  executor := frame.NewExecutor(WithTargetFPS(60))
  err := executor.Execute(ctx, gameLoop, afterFrame)
  // ..
}

func gameLoop() error {
  // run here your:
  // - world.Update
  // - world.Draw
  // - events handle
  // - etc..
  
  time.Sleep(time.Millisecond * 25) // emulate work..

  return nil
}

func afterFrame(stats frame.Stats) {
  // dt = stats.DeltaTime
  fmt.Printf("frm:%03d, FPS: %02d/%02d\n",
    stats.CurrentFrame,
    stats.CurrentFPS,
    stats.FrameTargetFPS,
  )
}

```

## Tasks

You can provide some minor tasks, that will be executed
only when we have free CPU time.

For example at `targetFPS=60`, our frame capacity is `16.6ms`
When your `gameLoop` took `10ms`, we have free `6.6ms` in current frame

This `6.6ms` will be used for tasks processing

### GC task

Super useful and required in most cases is __garbage collection__
task, that will process golang GC only when we have free time.

Of course on low-end CPU's, when our `FPS` always less that `targetFPS`,
it will be executed anyway, at least once in `10 seconds`

```go
executor := NewExecutor(
  WithTargetFPS(60),
  
  // add unlimited number of tasks
  WithTask(
    NewTask(
      func() {
        // try to run every frame
        // but not more often that once in 1s
        // but at least once in 10 second guaranteed
        runtime.GC()
        runtime.Gosched()
      },
      WithRunAtLeastOnceIn(time.Second * 10),
      WithRunAtMostOnceIn(time.Second),
      WithPriority(TaskPriorityLow),
    ),
  ),
)

executor.Execute( .. )
```

Also it already defined in lib, you can use default task:

```go
frame.NewDefaultTaskGarbageCollect()
```

### Custom tasks

You can add any number of tasks, and choose priority from `LOW` to `HIGH`,
also `Executor` will take into account other task properties like `LastRunTime`, `AvgExecutionTime`
and other in priority calculation.


## Available stats

```go
type Timings struct {
  StartAt  time.Time
  Duration time.Duration
}

type Stats struct {
  CurrentFrame uint64   // frameID since game start
  CurrentFPS   int      // real counted FPS
  DeltaTime    float64  // use it for all game calculations

  FrameFreeTime    time.Duration // how much was available
  FrameTargetFPS   int           // real fps ceil
  FramePossibleFPS int           // ~ calculated ceil
  FrameTimeLimit   time.Duration // 1s / targetFPS
  ThrottleTime     time.Duration // sleep time

  Execute Timings // full game time
  Frame   Timings // current frame time
  Process Timings // your `gameLoop` function time
  Tasks   Timings // tasks time
}
```

## Full Example

See code in [frame/executor_test](./frame/executor_test.go)

Test settings:
- TestTime  = 3s
- FPSLimit  = 24
- LogicTime = 25ms (4 frame per 100ms / 40 frames per second)

Tasks:
- high priority 5ms task (run at least once in second, but try to every 500ms)
- low priority GC (at least once in second, but try to every 100ms)

Test output:
```
| -- STATS --              | -- Frame --                               |
| elapsed | frame |  FPS   | capacity |       fn |    tasks | throttle |
|  0042ms |  001  |  24/24 |     41ms |     25ms |     06ms |     10ms |
|  0085ms |  002  |  24/24 |     41ms |     25ms |     00ms |     16ms |
|  0128ms |  003  |  24/24 |     41ms |     25ms |     00ms |     16ms |
|  0170ms |  004  |  24/24 |     41ms |     25ms |     01ms |     15ms |
|  0212ms |  005  |  24/24 |     41ms |     26ms |     00ms |     15ms |
|  0255ms |  006  |  24/24 |     41ms |     25ms |     00ms |     16ms |
|  0297ms |  007  |  24/24 |     41ms |     25ms |     00ms |     15ms |
|  0338ms |  008  |  24/24 |     41ms |     25ms |     00ms |     16ms |
|  0381ms |  009  |  24/24 |     41ms |     25ms |     00ms |     16ms |
|  0424ms |  010  |  24/24 |     41ms |     25ms |     01ms |     15ms |
|  0466ms |  011  |  24/24 |     41ms |     25ms |     00ms |     16ms |
|  0509ms |  012  |  24/24 |     41ms |     25ms |     00ms |     16ms |
|  0551ms |  013  |  24/24 |     41ms |     25ms |     05ms |     10ms |
|  0593ms |  014  |  24/24 |     41ms |     25ms |     00ms |     15ms |
|  0635ms |  015  |  24/24 |     41ms |     25ms |     00ms |     16ms |
|  0677ms |  016  |  24/24 |     41ms |     25ms |     00ms |     15ms |
|  0719ms |  017  |  24/24 |     41ms |     25ms |     00ms |     16ms |
|  0762ms |  018  |  24/24 |     41ms |     25ms |     00ms |     16ms |
|  0803ms |  019  |  24/24 |     41ms |     25ms |     00ms |     15ms |
|  0846ms |  020  |  24/24 |     41ms |     25ms |     00ms |     16ms |
|  0889ms |  021  |  24/24 |     41ms |     25ms |     00ms |     16ms |
|  0931ms |  022  |  24/24 |     41ms |     25ms |     00ms |     15ms |
|  0973ms |  023  |  24/24 |     41ms |     25ms |     00ms |     16ms |
|  1016ms |  024  |  24/24 |     41ms |     25ms |     00ms |     16ms |
|  1058ms |  025  |  24/24 |     41ms |     25ms |     05ms |     10ms |
|  1100ms |  026  |  24/24 |     41ms |     25ms |     00ms |     16ms |
|  1143ms |  027  |  24/24 |     41ms |     25ms |     00ms |     16ms |
|  1185ms |  028  |  24/24 |     41ms |     25ms |     00ms |     15ms |
|  1227ms |  029  |  24/24 |     41ms |     25ms |     00ms |     16ms |
|  1270ms |  030  |  24/24 |     41ms |     25ms |     00ms |     16ms |
|  1312ms |  031  |  24/24 |     41ms |     25ms |     00ms |     15ms |
|  1354ms |  032  |  24/24 |     41ms |     25ms |     00ms |     16ms |
|  1396ms |  033  |  24/24 |     41ms |     25ms |     00ms |     16ms |
|  1438ms |  034  |  24/24 |     41ms |     25ms |     00ms |     15ms |
|  1480ms |  035  |  24/24 |     41ms |     25ms |     00ms |     16ms |
|  1522ms |  036  |  24/24 |     41ms |     25ms |     00ms |     16ms |
|  1564ms |  037  |  24/24 |     41ms |     25ms |     06ms |     10ms |
|  1606ms |  038  |  24/24 |     41ms |     25ms |     00ms |     15ms |
|  1649ms |  039  |  24/24 |     41ms |     25ms |     00ms |     16ms |
|  1691ms |  040  |  24/24 |     41ms |     25ms |     00ms |     15ms |
|  1733ms |  041  |  24/24 |     41ms |     25ms |     00ms |     15ms |
|  1775ms |  042  |  24/24 |     41ms |     25ms |     00ms |     16ms |
|  1817ms |  043  |  24/24 |     41ms |     25ms |     00ms |     15ms |
|  1860ms |  044  |  24/24 |     41ms |     25ms |     00ms |     15ms |
|  1902ms |  045  |  24/24 |     41ms |     25ms |     00ms |     16ms |
|  1944ms |  046  |  24/24 |     41ms |     25ms |     00ms |     15ms |
|  1986ms |  047  |  24/24 |     41ms |     25ms |     00ms |     15ms |
|  2029ms |  048  |  24/24 |     41ms |     25ms |     00ms |     16ms |
|  2070ms |  049  |  24/24 |     41ms |     25ms |     05ms |     10ms |
|  2113ms |  050  |  24/24 |     41ms |     25ms |     00ms |     15ms |
|  2155ms |  051  |  24/24 |     41ms |     25ms |     00ms |     16ms |
|  2197ms |  052  |  24/24 |     41ms |     25ms |     00ms |     16ms |
|  2240ms |  053  |  24/24 |     41ms |     25ms |     00ms |     16ms |
|  2282ms |  054  |  24/24 |     41ms |     25ms |     00ms |     16ms |
|  2325ms |  055  |  24/24 |     41ms |     25ms |     00ms |     15ms |
|  2367ms |  056  |  24/24 |     41ms |     25ms |     00ms |     16ms |
|  2410ms |  057  |  24/24 |     41ms |     25ms |     00ms |     16ms |
|  2452ms |  058  |  24/24 |     41ms |     25ms |     00ms |     15ms |
|  2494ms |  059  |  24/24 |     41ms |     25ms |     00ms |     16ms |
|  2537ms |  060  |  24/24 |     41ms |     25ms |     00ms |     16ms |
|  2579ms |  061  |  24/24 |     41ms |     25ms |     05ms |     10ms |
|  2621ms |  062  |  24/24 |     41ms |     25ms |     00ms |     15ms |
|  2663ms |  063  |  24/24 |     41ms |     25ms |     00ms |     16ms |
|  2706ms |  064  |  24/24 |     41ms |     25ms |     00ms |     16ms |
|  2748ms |  065  |  24/24 |     41ms |     25ms |     00ms |     16ms |
|  2791ms |  066  |  24/24 |     41ms |     25ms |     00ms |     16ms |
|  2833ms |  067  |  24/24 |     41ms |     25ms |     00ms |     15ms |
|  2874ms |  068  |  24/24 |     41ms |     25ms |     00ms |     16ms |
|  2917ms |  069  |  24/24 |     41ms |     25ms |     00ms |     16ms |
|  2959ms |  070  |  24/24 |     41ms |     25ms |     00ms |     15ms |
|  3001ms |  071  |  24/24 |     41ms |     25ms |     00ms |     15ms |
--- PASS: TestExecutor_Execute (3.00s)
```
