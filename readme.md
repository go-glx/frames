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
| -- STATS --                     | -- Frame --                               |
| elapsed | frame |  FPS  |  TPS  | capacity |       fn |    fixed |    tasks | throttle |
|  0045ms |  001  | 24/24 | 06/06 |     41ms |     04ms |     00ms |     06ms |     35ms |
|  0086ms |  002  | 24/24 | 06/06 |     41ms |     04ms |     00ms |     00ms |     36ms |
|  0129ms |  003  | 24/24 | 06/06 |     41ms |     04ms |     00ms |     00ms |     38ms |
|  0177ms |  004  | 24/24 | 06/06 |     41ms |     04ms |     10ms |     00ms |     42ms |
|  0212ms |  005  | 24/24 | 06/06 |     41ms |     04ms |     00ms |     00ms |     31ms |
|  0255ms |  006  | 24/24 | 06/06 |     41ms |     04ms |     00ms |     00ms |     38ms |
|  0296ms |  007  | 24/24 | 06/06 |     41ms |     04ms |     00ms |     00ms |     35ms |
|  0344ms |  008  | 24/24 | 06/06 |     41ms |     04ms |     10ms |     00ms |     43ms |
|  0379ms |  009  | 24/24 | 06/06 |     41ms |     04ms |     00ms |     00ms |     30ms |
|  0421ms |  010  | 24/24 | 06/06 |     41ms |     04ms |     00ms |     00ms |     37ms |
|  0463ms |  011  | 24/24 | 06/06 |     41ms |     04ms |     00ms |     00ms |     37ms |
|  0510ms |  012  | 24/24 | 06/06 |     41ms |     04ms |     10ms |     00ms |     42ms |
|  0544ms |  013  | 24/24 | 06/06 |     41ms |     04ms |     00ms |     05ms |     24ms |
|  0589ms |  014  | 24/24 | 06/06 |     41ms |     04ms |     00ms |     00ms |     40ms |
|  0630ms |  015  | 24/24 | 06/06 |     41ms |     04ms |     00ms |     00ms |     36ms |
|  0677ms |  016  | 24/24 | 06/06 |     41ms |     04ms |     10ms |     00ms |     41ms |
|  0712ms |  017  | 24/24 | 06/06 |     41ms |     04ms |     00ms |     00ms |     30ms |
|  0755ms |  018  | 24/24 | 06/06 |     41ms |     04ms |     00ms |     00ms |     38ms |
|  0796ms |  019  | 24/24 | 06/06 |     41ms |     04ms |     00ms |     00ms |     36ms |
|  0844ms |  020  | 24/24 | 06/06 |     41ms |     04ms |     10ms |     00ms |     43ms |
|  0880ms |  021  | 24/24 | 06/06 |     41ms |     04ms |     00ms |     00ms |     31ms |
|  0920ms |  022  | 24/24 | 06/06 |     41ms |     04ms |     00ms |     00ms |     36ms |
|  0963ms |  023  | 24/24 | 06/06 |     41ms |     04ms |     00ms |     00ms |     38ms |
|  1011ms |  024  | 23/24 | 05/06 |     41ms |     04ms |     10ms |     00ms |     43ms |
|  1047ms |  025  | 23/24 | 05/06 |     41ms |     04ms |     00ms |     06ms |     25ms |
|  1089ms |  026  | 23/24 | 05/06 |     41ms |     04ms |     00ms |     00ms |     38ms |
|  1129ms |  027  | 23/24 | 05/06 |     41ms |     04ms |     00ms |     00ms |     35ms |
|  1176ms |  028  | 23/24 | 05/06 |     41ms |     04ms |     10ms |     00ms |     42ms |
|  1213ms |  029  | 23/24 | 05/06 |     41ms |     04ms |     00ms |     00ms |     31ms |
|  1256ms |  030  | 23/24 | 05/06 |     41ms |     04ms |     00ms |     00ms |     38ms |
|  1297ms |  031  | 23/24 | 05/06 |     41ms |     04ms |     00ms |     00ms |     36ms |
|  1344ms |  032  | 23/24 | 05/06 |     41ms |     04ms |     10ms |     00ms |     42ms |
|  1380ms |  033  | 23/24 | 05/06 |     41ms |     04ms |     00ms |     00ms |     31ms |
|  1423ms |  034  | 23/24 | 05/06 |     41ms |     04ms |     00ms |     00ms |     37ms |
|  1465ms |  035  | 23/24 | 05/06 |     41ms |     04ms |     00ms |     00ms |     37ms |
|  1510ms |  036  | 23/24 | 05/06 |     41ms |     04ms |     10ms |     00ms |     41ms |
|  1546ms |  037  | 23/24 | 05/06 |     41ms |     04ms |     00ms |     00ms |     30ms |
|  1589ms |  038  | 23/24 | 05/06 |     41ms |     04ms |     00ms |     05ms |     32ms |
|  1630ms |  039  | 23/24 | 05/06 |     41ms |     04ms |     00ms |     00ms |     37ms |
|  1676ms |  040  | 23/24 | 05/06 |     41ms |     04ms |     10ms |     00ms |     41ms |
|  1715ms |  041  | 23/24 | 05/06 |     41ms |     04ms |     00ms |     00ms |     34ms |
|  1755ms |  042  | 23/24 | 05/06 |     41ms |     04ms |     00ms |     00ms |     36ms |
|  1798ms |  043  | 23/24 | 05/06 |     41ms |     04ms |     00ms |     00ms |     37ms |
|  1844ms |  044  | 23/24 | 05/06 |     41ms |     04ms |     10ms |     00ms |     42ms |
|  1880ms |  045  | 23/24 | 05/06 |     41ms |     04ms |     00ms |     00ms |     31ms |
|  1924ms |  046  | 23/24 | 05/06 |     41ms |     04ms |     00ms |     00ms |     38ms |
|  1963ms |  047  | 23/24 | 05/06 |     41ms |     04ms |     00ms |     00ms |     34ms |
|  2011ms |  048  | 23/24 | 05/06 |     41ms |     04ms |     10ms |     00ms |     42ms |
|  2049ms |  049  | 25/24 | 07/06 |     41ms |     04ms |     00ms |     00ms |     33ms |
|  2089ms |  050  | 25/24 | 07/06 |     41ms |     04ms |     00ms |     05ms |     30ms |
|  2132ms |  051  | 25/24 | 07/06 |     41ms |     04ms |     00ms |     00ms |     38ms |
|  2177ms |  052  | 25/24 | 07/06 |     41ms |     04ms |     10ms |     00ms |     40ms |
|  2214ms |  053  | 25/24 | 07/06 |     41ms |     04ms |     00ms |     00ms |     32ms |
|  2256ms |  054  | 25/24 | 07/06 |     41ms |     04ms |     00ms |     00ms |     37ms |
|  2299ms |  055  | 25/24 | 07/06 |     41ms |     04ms |     00ms |     00ms |     37ms |
|  2343ms |  056  | 25/24 | 07/06 |     41ms |     04ms |     10ms |     00ms |     40ms |
|  2380ms |  057  | 25/24 | 07/06 |     41ms |     04ms |     00ms |     00ms |     32ms |
|  2423ms |  058  | 25/24 | 07/06 |     41ms |     04ms |     00ms |     00ms |     38ms |
|  2466ms |  059  | 25/24 | 07/06 |     41ms |     04ms |     00ms |     00ms |     38ms |
|  2511ms |  060  | 25/24 | 07/06 |     41ms |     04ms |     10ms |     00ms |     40ms |
|  2549ms |  061  | 25/24 | 07/06 |     41ms |     04ms |     00ms |     00ms |     32ms |
|  2590ms |  062  | 25/24 | 07/06 |     41ms |     04ms |     00ms |     00ms |     36ms |
|  2631ms |  063  | 25/24 | 07/06 |     41ms |     04ms |     00ms |     05ms |     31ms |
|  2677ms |  064  | 25/24 | 07/06 |     41ms |     04ms |     10ms |     00ms |     41ms |
|  2716ms |  065  | 25/24 | 07/06 |     41ms |     04ms |     00ms |     00ms |     34ms |
|  2758ms |  066  | 25/24 | 07/06 |     41ms |     04ms |     00ms |     00ms |     37ms |
|  2798ms |  067  | 25/24 | 07/06 |     41ms |     04ms |     00ms |     00ms |     35ms |
|  2844ms |  068  | 25/24 | 07/06 |     41ms |     04ms |     10ms |     00ms |     42ms |
|  2881ms |  069  | 25/24 | 07/06 |     41ms |     04ms |     00ms |     00ms |     32ms |
|  2924ms |  070  | 25/24 | 07/06 |     41ms |     04ms |     00ms |     00ms |     38ms |
|  2967ms |  071  | 25/24 | 07/06 |     41ms |     04ms |     00ms |     00ms |     38ms |
--- PASS: TestExecutor_Execute (3.00s)
```
