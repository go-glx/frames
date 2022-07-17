package frame

import (
	"context"
	"fmt"
	"math/rand"
	"path"
	"testing"
	"time"

	"github.com/fogleman/gg"
	"github.com/stretchr/testify/assert"
)

const testTraceOutDirectory = "./../example/trace"

type testTraceBlockType uint8

const (
	testTraceBlockTick testTraceBlockType = iota
	testTraceBlockFrame
	testTraceBlockTask
)

type testMeasure struct {
	bType   testTraceBlockType
	startAt time.Time
	endAt   time.Time
}

func testMeasureFunction(bType testTraceBlockType, fn func()) testMeasure {
	start := time.Now()
	fn()
	end := time.Now()

	return testMeasure{
		bType:   bType,
		startAt: start,
		endAt:   end,
	}
}

type testTraceVariant struct {
	outputName           string
	testDuration         time.Duration
	targetTicksPerSecond int
	latencyTick          time.Duration
	latencyFrame         time.Duration

	additionalLogicFrame func(frameID int)
	additionalLogicTick  func(tickID int)
	additionalTasks      []*Task

	debugFrameTimings bool
}

func testTraceVariants() []testTraceVariant {
	return []testTraceVariant{
		{
			outputName:           "example",
			testDuration:         testExampleDuration,
			targetTicksPerSecond: testExampleTicksRate,

			latencyTick:  testExampleLatencyTick,
			latencyFrame: testExampleLatencyFrame,

			additionalTasks: []*Task{
				testExampleTask1(testExampleLatencyTask),
				testExampleTaskGC(),
			},
		},
		{
			outputName:           "1_60tps",
			testDuration:         time.Second * 1,
			targetTicksPerSecond: 60,

			latencyTick:  time.Millisecond * 5,
			latencyFrame: time.Millisecond * 9,

			additionalLogicFrame: func(frameID int) {
				time.Sleep(time.Millisecond * time.Duration(rand.Float64()*33))
			},

			additionalTasks: []*Task{
				NewTask(func() {
					time.Sleep(time.Millisecond * 5)
				},
					WithPriority(TaskPriorityHigh),
					WithRunAtLeastOnceIn(time.Millisecond*100),
				),
			},
		},
		// {
		// 	outputName: "2_30fps_task5ms",
		// 	outputDescriptionMd: `
		// 		Test have only frames logic:
		// 		- every frame have 20ms latency.
		// 		- frames from 1 to 10 - have additional 10ms latency
		// 		- frame #3 will emulate super lag (+100ms latency)
		//
		// 		Additionally this configuration contains tasks:
		// 		- #0: This test contains standard golang GC task
		// 		- #1: Run at least once per 100ms, but at most one time in 50ms. Task will emulate 5ms of work
		// 	`,
		// 	testDuration:          time.Second * 1,
		// 	targetFramesPerSecond: 30,
		// 	targetTicksPerSecond:  0,
		//
		// 	// 33.3 shared budget
		// 	latencyFrame: time.Millisecond * 20,
		// 	latencyTick:  time.Millisecond * 0,
		//
		// 	additionalLogicFrame: func(frameID int) {
		// 		if frameID == 3 {
		// 			time.Sleep(time.Millisecond * 100)
		// 		}
		//
		// 		if frameID < 10 {
		// 			// emulate lag at 10 frame
		// 			time.Sleep(time.Millisecond * 10)
		// 		}
		// 	},
		//
		// 	additionalTasks: []*Task{
		// 		NewTask(func() {
		// 			time.Sleep(time.Millisecond * 5)
		// 		},
		// 			WithRunAtLeastOnceIn(time.Millisecond*100),
		// 			WithRunAtMostOnceIn(time.Millisecond*50),
		// 			WithPriority(TaskPriorityHigh),
		// 		),
		// 	},
		// },
		// {
		// 	outputName: "3_30fps_60tps",
		// 	outputDescriptionMd: `
		// 		Simple test with both systems active:
		// 		- frames (FPS) with target at 120fps
		// 		- ticks (TPS) with target at 30tps
		//
		// 		"Ticks" is fixed/physics/stable update in term of other game engines
		// 		Ticks have more priority to run, and this will degrade frames performance
		// 		when not have enough CPU power to process ticks
		//
		// 		Test rules:
		// 		- avg latency on frame = 4ms
		// 		- avg latency on tick = 2ms
		// 		- frame 5..10 will emulate lag +5ms
		// 		- tick 15..20 will emulate lag +5ms
		// 		- all 25..30 will emulate full lag +5ms
		// 	`,
		// 	testDuration:          time.Second * 1,
		// 	targetFramesPerSecond: 120,
		// 	targetTicksPerSecond:  30,
		//
		// 	latencyFrame: time.Millisecond * 4,
		// 	latencyTick:  time.Millisecond * 2,
		//
		// 	additionalLogicFrame: func(frameID int) {
		// 		if frameID >= 5 && frameID <= 10 {
		// 			time.Sleep(time.Millisecond * 5)
		// 		}
		// 		if frameID >= 25 && frameID <= 30 {
		// 			time.Sleep(time.Millisecond * 5)
		// 		}
		// 	},
		//
		// 	additionalLogicTick: func(tickID int) {
		// 		if tickID >= 15 && tickID <= 20 {
		// 			time.Sleep(time.Millisecond * 5)
		// 		}
		// 		if tickID >= 25 && tickID <= 30 {
		// 			time.Sleep(time.Millisecond * 5)
		// 		}
		// 	},
		// },
	}
}

func TestTraceExecutor(t *testing.T) {
	for _, variant := range testTraceVariants() {
		ctx, cancel := context.WithTimeout(context.Background(), variant.testDuration)

		collectedStats := make([]Stats, 0)
		statsCollector := func(s Stats) {
			collectedStats = append(collectedStats, s)
		}

		inits := make([]ExecutorInitializer, 0)
		inits = append(inits, WithTargetTPS(variant.targetTicksPerSecond))
		inits = append(inits, WithTask(NewDefaultTaskGarbageCollect()))
		inits = append(inits, WithStatsCollector(statsCollector))

		for _, task := range variant.additionalTasks {
			inits = append(inits, WithTask(task))
		}

		testExecutor := NewExecutor(inits...)

		measures := make([]testMeasure, 0)

		currentTickID := 0
		fnUpdate := func(stats TickStats) error {
			currentTickID++
			measures = append(measures, testMeasureFunction(testTraceBlockTick, func() {
				time.Sleep(variant.latencyTick)

				if variant.additionalLogicTick != nil {
					variant.additionalLogicTick(currentTickID)
				}
			}))
			return nil
		}

		currentFrameID := 0
		fnDraw := func() error {
			currentFrameID++
			measures = append(measures, testMeasureFunction(testTraceBlockFrame, func() {
				time.Sleep(variant.latencyFrame)

				if variant.additionalLogicFrame != nil {
					variant.additionalLogicFrame(currentFrameID)
				}
			}))

			return nil
		}

		err := testExecutor.Execute(ctx, fnUpdate, fnDraw)
		assert.NoError(t, err)

		cancel()

		if variant.debugFrameTimings {
			for _, stat := range collectedStats {
				fmt.Printf("%d.\n"+
					"- tick start: %dus\n"+
					"-   tick end: %dus\n"+
					"-  frm start: %dus\n"+
					"-    frm end: %dus\n",

					stat.CycleID,
					stat.Tick.Start.Sub(stat.Game.Start).Microseconds(),
					(stat.Tick.Start.Sub(stat.Game.Start) + stat.Tick.Duration).Microseconds(),
					stat.Frame.Start.Sub(stat.Game.Start).Microseconds(),
					(stat.Frame.Start.Sub(stat.Game.Start) + stat.Tick.Duration).Microseconds(),
				)
			}
		}

		testOutput(t, testExecutor, variant, measures, collectedStats)
	}
}

func testOutput(t *testing.T, exec *Executor, variant testTraceVariant, measures []testMeasure, stats []Stats) {
	// colors
	// https://coolors.co/palette/355070-6d597a-b56576-e56b6f-eaac8b
	// https://coolors.co/palette/ff595e-ffca3a-8ac926-1982c4-6a4c93
	const (
		colBack                 = "#ffffff00" // "#fff"
		colText                 = "#ef6351"   // "#001"
		colTimeline             = "#6F5E53"   // "#000"
		colTimelineStrokeSecond = "#AB947E"   // "#111"
		colTimelineStrokeHalf   = "#8A7968"   // "#333"
		colTimelineStroke100ms  = "#8A7968"   // "#555"
		colTimelineStrokeBudget = "#593D3B"   // "#999"
		colBlockFrame           = "#FFCA3A"   // "#e40"
		colBlockTick            = "#8AC926"   // "#0f3"
		colBlockTask            = "#1982C4"   // "#02e"
	)

	// rules
	const (
		widthPxPerSecond = float64(2000)
		widthPxPerUs     = widthPxPerSecond / 1_000_000
		sampleHeight     = float64(50)
		mainPaddingX     = float64(20)
		mainPaddingY     = float64(40)
		timeLineMargin   = float64(15)
		infoHeight       = float64(25)
	)

	// calculate graph size
	timeLineDurationUs := float64(exec.stats.Game.Duration.Microseconds())
	timelineWidth := timeLineDurationUs * widthPxPerUs
	fullWidth := (mainPaddingX * 2) + timelineWidth
	timelineY := mainPaddingY + infoHeight + sampleHeight + timeLineMargin
	fullHeight := timelineY + timeLineMargin + mainPaddingY

	// canvas
	dc := gg.NewContext(int(fullWidth), int(fullHeight))

	// bg
	dc.SetHexColor(colBack)
	dc.Clear()

	// top info
	dc.SetHexColor(colText)
	infoText := fmt.Sprintf("Tick: { lat:%dms, target: %d/s } Frame: { lat:%dms }",
		variant.latencyTick.Milliseconds(),
		variant.targetTicksPerSecond,
		variant.latencyFrame.Milliseconds(),
	)
	dc.DrawStringAnchored(infoText, mainPaddingX, 15, 0, 0)

	// timeline
	dc.SetHexColor(colTimeline)
	dc.DrawLine(mainPaddingX, timelineY, mainPaddingX+timelineWidth, timelineY)
	dc.Stroke()

	// timeline strokes
	drawStroke := func(interval time.Duration, color string, halfHeight float64, withText bool) {
		curTime := time.Microsecond * 0
		for x := mainPaddingX; x <= timelineWidth; x += float64(interval.Microseconds()) * widthPxPerUs {
			dc.SetHexColor(color)
			dc.DrawLine(x, timelineY-halfHeight, x, timelineY+halfHeight)
			if halfHeight >= 10 {
				// big line
				dc.SetLineWidth(2)
			}

			dc.Stroke()

			if withText {
				curTimeText := fmt.Sprintf("%dms", curTime.Milliseconds())
				dc.DrawStringAnchored(curTimeText, x, timelineY+halfHeight+5, 0.5, 0.5)
			}

			curTime += interval
		}
	}

	drawStroke(time.Second, colTimelineStrokeSecond, 10, false)
	drawStroke(time.Millisecond*500, colTimelineStrokeHalf, 8, false)
	drawStroke(time.Millisecond*100, colTimelineStroke100ms, 4, true)
	drawStroke(exec.stats.Rate, colTimelineStrokeBudget, 1, false)

	// blocks
	drawBlocks := func(samples []testMeasure, color string) {
		for ind, sample := range samples {
			relativeStartAt := sample.startAt.Sub(exec.stats.Game.Start)
			x := mainPaddingX + (float64(relativeStartAt.Microseconds()) * widthPxPerUs)
			y := timelineY - timeLineMargin - sampleHeight
			width := float64(sample.endAt.Sub(sample.startAt).Microseconds()) * widthPxPerUs

			dc.SetHexColor(color)
			dc.DrawRectangle(x, y, width, sampleHeight)
			dc.Fill()

			if sample.bType == testTraceBlockTick {
				dc.DrawStringAnchored(fmt.Sprintf("%d", ind+1), x+(width/2), y-timeLineMargin, 0.5, 0)
			}
		}
	}

	tasks := transformToMeasure(stats, func(s Stats) *testMeasure {
		if s.Tasks.Duration <= time.Microsecond*50 {
			return nil
		}

		return &testMeasure{
			bType:   testTraceBlockTask,
			startAt: s.Tasks.Start,
			endAt:   s.Tasks.Start.Add(s.Tasks.Duration),
		}
	})

	drawBlocks(filterSample(measures, testTraceBlockFrame), colBlockFrame)
	drawBlocks(filterSample(measures, testTraceBlockTick), colBlockTick)
	drawBlocks(tasks, colBlockTask)

	// output
	outputPath := path.Join(testTraceOutDirectory, fmt.Sprintf("%s.png", variant.outputName))
	err := dc.SavePNG(outputPath)
	assert.NoError(t, err)
}

func transformToMeasure(all []Stats, transform func(s Stats) *testMeasure) []testMeasure {
	list := make([]testMeasure, 0)

	for _, stat := range all {
		sample := transform(stat)
		if sample == nil {
			continue
		}

		list = append(list, *sample)
	}

	return list
}

func filterSample(all []testMeasure, sType testTraceBlockType) []testMeasure {
	list := make([]testMeasure, 0)

	for _, measure := range all {
		if measure.bType != sType {
			continue
		}

		list = append(list, measure)
	}

	return list
}
