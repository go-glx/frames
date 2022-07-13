# go-glx / frames

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
  executor := frame.NewExecutor(WithTargetFPS(60), WithTargetTPS(50))
  err := executor.Execute(ctx, update, fixedUpdate, afterFrame)
  // ..
}

func update() error {
  // configure: WithTargetFPS(60) (frames per second)
  // run here your:
  // - world.Update
  // - world.Draw
  // - events handle
  // - etc..
  
  time.Sleep(time.Millisecond * 25) // emulate work..

  return nil
}

func fixedUpdate() error {
  // configure: WithTargetTPS(50) (ticks per second)
  // fixed update (game tick)
  // useful for physics calculations
  // or other fixed game loop updates
  
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
  // DeltaTime is current frame duration in seconds.
  //
  // This useful for integrate into game logic update step
  // for example if you want to move Player at 100 units per second, you can do something like that:
  //    player.x += 100*DeltaTime
  //    - If we have 60fps, DeltaTime=1s/60 = 0.0166666s (16ms)
  //    - If we have 30fps, DeltaTime=1s/30 = 0.0333333s (33ms)
  //
  // So:
  //    - (100*0.0166666) * 60 = 100
  //    - (100*0.0333333) * 30 = 100
  //
  // Typically game have unstable FPS, but player anyway will be moved at 100px per second, not depend on current rate
  // When cpu power > needed, this always will be "1s / FrameTargetFPS", because of throttling (see ThrottleTime)
  DeltaTime float64

  CurrentFrame uint64 // global frame ID since game start, every frame will increment this number on +1
  CurrentTPS   int    // real counted ticks per second (ticks is fixed/physics update)
  CurrentFPS   int    // real counted frames per second

  FrameFreeTime    time.Duration // how much free time we had in current frame (most likely was spent to tasks, GC, etc..)
  FrameTargetTPS   int           // target ticks per second (or fixed/physics update). Default is 50tps, but can be configured to any number
  FrameTargetFPS   int           // target frames per second. Default is 60fps, can be configured to any number
  FramePossibleFPS int           // maximum calculated FPS that can be theoretically achieved in current CPU
  FrameTimeLimit   time.Duration // how much time we have for processing in every frame. "= 1s / FrameTargetFPS"
  ThrottleTime     time.Duration // when CPU is more powerful when we need for processing at FrameTargetFPS rate, frame will sleep ThrottleTime in end of current frame

  Execute Timings // how long game is running
  Frame   Timings // current frame stats (process + fixes + tasks)
  Process Timings // timings for main update function
  Fixed   Timings // timings for fixed update function
  Tasks   Timings // timings for additional tasks that was executed in this frame
}
```

## Full Example

See code in [frame/executor_test](./frame/executor_test.go)

Test settings:
- TestTime  = 3s
- FPSLimit  = 24
- TPSLimit  = 6
- LogicTime = 25ms (4 frame per 100ms / 40 frames per second)
- TickTime = 10ms (10 frame per 100ms / 100 frames per second)

Tasks:
- high priority 5ms task (run at least once in second, but try to every 500ms)
- low priority GC (at least once in second, but try to every 100ms)

Test output:
```
| -- STATS --                     | -- Frame --                                          |
| elapsed | frame |  FPS  |  TPS  | capacity |       fn |    fixed |    tasks | throttle |
|  0042ms |  001  | 24/24 | 06/06 |     41ms |     25ms |     00ms |     05ms |     11ms |
|  0085ms |  002  | 24/24 | 06/06 |     41ms |     25ms |     00ms |     00ms |     17ms |
|  0127ms |  003  | 24/24 | 06/06 |     41ms |     25ms |     00ms |     00ms |     16ms |
|  0177ms |  004  | 24/24 | 06/06 |     41ms |     25ms |     10ms |     01ms |     14ms |
|  0220ms |  005  | 24/24 | 06/06 |     41ms |     25ms |     00ms |     00ms |     17ms |
|  0261ms |  006  | 24/24 | 06/06 |     41ms |     25ms |     00ms |     00ms |     15ms |
|  0303ms |  007  | 24/24 | 06/06 |     41ms |     25ms |     00ms |     00ms |     16ms |
|  0343ms |  008  | 24/24 | 06/06 |     41ms |     25ms |     10ms |     00ms |     15ms |
|  0386ms |  009  | 24/24 | 06/06 |     41ms |     25ms |     00ms |     00ms |     17ms |
|  0427ms |  010  | 24/24 | 06/06 |     41ms |     25ms |     00ms |     00ms |     16ms |
|  0470ms |  011  | 24/24 | 06/06 |     41ms |     25ms |     00ms |     00ms |     17ms |
|  0510ms |  012  | 24/24 | 06/06 |     41ms |     25ms |     10ms |     00ms |     15ms |
|  0552ms |  013  | 24/24 | 06/06 |     41ms |     25ms |     00ms |     05ms |     10ms |
|  0594ms |  014  | 24/24 | 06/06 |     41ms |     25ms |     00ms |     00ms |     16ms |
|  0637ms |  015  | 24/24 | 06/06 |     41ms |     25ms |     00ms |     00ms |     17ms |
|  0678ms |  016  | 24/24 | 06/06 |     41ms |     25ms |     10ms |     01ms |     13ms |
|  0719ms |  017  | 24/24 | 06/06 |     41ms |     25ms |     00ms |     00ms |     16ms |
|  0761ms |  018  | 24/24 | 06/06 |     41ms |     25ms |     00ms |     00ms |     16ms |
|  0803ms |  019  | 24/24 | 06/06 |     41ms |     25ms |     00ms |     00ms |     16ms |
|  0844ms |  020  | 24/24 | 06/06 |     41ms |     25ms |     10ms |     00ms |     15ms |
|  0887ms |  021  | 24/24 | 06/06 |     41ms |     25ms |     00ms |     00ms |     16ms |
|  0928ms |  022  | 24/24 | 06/06 |     41ms |     25ms |     00ms |     00ms |     15ms |
|  0970ms |  023  | 24/24 | 06/06 |     41ms |     25ms |     00ms |     00ms |     16ms |
|  1011ms |  024  | 23/24 | 06/06 |     41ms |     25ms |     10ms |     00ms |     15ms |
|  1053ms |  025  | 23/24 | 06/06 |     41ms |     25ms |     00ms |     05ms |     11ms |
|  1095ms |  026  | 23/24 | 06/06 |     41ms |     25ms |     00ms |     00ms |     16ms |
|  1137ms |  027  | 23/24 | 06/06 |     41ms |     25ms |     00ms |     00ms |     16ms |
|  1177ms |  028  | 23/24 | 06/06 |     41ms |     25ms |     10ms |     00ms |     13ms |
|  1221ms |  029  | 23/24 | 06/06 |     41ms |     25ms |     00ms |     00ms |     18ms |
|  1262ms |  030  | 23/24 | 06/06 |     41ms |     25ms |     00ms |     00ms |     15ms |
|  1305ms |  031  | 23/24 | 06/06 |     41ms |     25ms |     00ms |     00ms |     17ms |
|  1344ms |  032  | 23/24 | 06/06 |     41ms |     25ms |     10ms |     00ms |     14ms |
|  1387ms |  033  | 23/24 | 06/06 |     41ms |     25ms |     00ms |     00ms |     17ms |
|  1429ms |  034  | 23/24 | 06/06 |     41ms |     25ms |     00ms |     00ms |     16ms |
|  1472ms |  035  | 23/24 | 06/06 |     41ms |     25ms |     00ms |     00ms |     17ms |
|  1511ms |  036  | 23/24 | 06/06 |     41ms |     25ms |     10ms |     00ms |     13ms |
|  1554ms |  037  | 23/24 | 06/06 |     41ms |     25ms |     00ms |     06ms |     11ms |
|  1595ms |  038  | 23/24 | 06/06 |     41ms |     25ms |     00ms |     00ms |     16ms |
|  1637ms |  039  | 23/24 | 06/06 |     41ms |     25ms |     00ms |     00ms |     16ms |
|  1678ms |  040  | 23/24 | 06/06 |     41ms |     25ms |     10ms |     00ms |     15ms |
|  1721ms |  041  | 23/24 | 06/06 |     41ms |     25ms |     00ms |     00ms |     16ms |
|  1762ms |  042  | 23/24 | 06/06 |     41ms |     25ms |     00ms |     00ms |     16ms |
|  1804ms |  043  | 23/24 | 06/06 |     41ms |     25ms |     00ms |     00ms |     16ms |
|  1844ms |  044  | 23/24 | 06/06 |     41ms |     25ms |     10ms |     00ms |     15ms |
|  1888ms |  045  | 23/24 | 06/06 |     41ms |     25ms |     00ms |     00ms |     18ms |
|  1930ms |  046  | 23/24 | 06/06 |     41ms |     25ms |     00ms |     00ms |     16ms |
|  1971ms |  047  | 23/24 | 06/06 |     41ms |     25ms |     00ms |     00ms |     15ms |
|  2012ms |  048  | 24/24 | 06/06 |     41ms |     25ms |     10ms |     00ms |     16ms |
|  2055ms |  049  | 24/24 | 06/06 |     41ms |     25ms |     00ms |     05ms |     11ms |
|  2097ms |  050  | 24/24 | 06/06 |     41ms |     25ms |     00ms |     00ms |     16ms |
|  2139ms |  051  | 24/24 | 06/06 |     41ms |     25ms |     00ms |     00ms |     16ms |
|  2177ms |  052  | 24/24 | 06/06 |     41ms |     25ms |     10ms |     00ms |     12ms |
|  2221ms |  053  | 24/24 | 06/06 |     41ms |     25ms |     00ms |     00ms |     18ms |
|  2263ms |  054  | 24/24 | 06/06 |     41ms |     25ms |     00ms |     00ms |     16ms |
|  2305ms |  055  | 24/24 | 06/06 |     41ms |     25ms |     00ms |     00ms |     15ms |
|  2345ms |  056  | 24/24 | 06/06 |     41ms |     25ms |     10ms |     00ms |     14ms |
|  2388ms |  057  | 24/24 | 06/06 |     41ms |     25ms |     00ms |     00ms |     17ms |
|  2430ms |  058  | 24/24 | 06/06 |     41ms |     25ms |     00ms |     00ms |     16ms |
|  2472ms |  059  | 24/24 | 06/06 |     41ms |     25ms |     00ms |     00ms |     16ms |
|  2511ms |  060  | 24/24 | 06/06 |     41ms |     25ms |     10ms |     00ms |     14ms |
|  2555ms |  061  | 24/24 | 06/06 |     41ms |     25ms |     00ms |     00ms |     18ms |
|  2596ms |  062  | 24/24 | 06/06 |     41ms |     25ms |     00ms |     05ms |     10ms |
|  2639ms |  063  | 24/24 | 06/06 |     41ms |     25ms |     00ms |     00ms |     17ms |
|  2679ms |  064  | 24/24 | 06/06 |     41ms |     25ms |     10ms |     00ms |     13ms |
|  2723ms |  065  | 24/24 | 06/06 |     41ms |     25ms |     00ms |     00ms |     18ms |
|  2764ms |  066  | 24/24 | 06/06 |     41ms |     25ms |     00ms |     00ms |     15ms |
|  2805ms |  067  | 24/24 | 06/06 |     41ms |     25ms |     00ms |     00ms |     15ms |
|  2845ms |  068  | 24/24 | 06/06 |     41ms |     25ms |     10ms |     00ms |     14ms |
|  2889ms |  069  | 24/24 | 06/06 |     41ms |     25ms |     00ms |     00ms |     19ms |
|  2930ms |  070  | 24/24 | 06/06 |     41ms |     25ms |     00ms |     00ms |     14ms |
|  2973ms |  071  | 24/24 | 06/06 |     41ms |     25ms |     00ms |     00ms |     16ms |
--- PASS: TestExecutor_Execute (3.00s)
```
