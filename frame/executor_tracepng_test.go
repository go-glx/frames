package frame

import (
	"context"
	"fmt"
	"path"
	"testing"
	"time"

	"github.com/fogleman/gg"
	"github.com/stretchr/testify/assert"
)

const testTraceOutDirectory = "./../example/trace"

type testTraceBlockType uint8

const (
	testTraceBlockUnknown testTraceBlockType = iota
	testTraceBlockThrottle
	testTraceBlockFrame
	testTraceBlockTick
	testTraceBlockSync
	testTraceBlockTask
)

type testMeasure struct {
	bType   testTraceBlockType
	startAt time.Time
	endAt   time.Time
}

func waitMeasure(sleepTime time.Duration, bType testTraceBlockType) testMeasure {
	start := time.Now()
	time.Sleep(sleepTime)
	end := time.Now()

	return testMeasure{
		bType:   bType,
		startAt: start,
		endAt:   end,
	}
}

type testTraceVariant struct {
	outputName            string
	testDuration          time.Duration
	targetFramesPerSecond int
	targetTicksPerSecond  int
	latencyFrame          time.Duration
	latencyTick           time.Duration
}

func testTraceVariants() []testTraceVariant {
	return []testTraceVariant{
		{
			outputName:            "simple",
			testDuration:          time.Second * 1,
			targetFramesPerSecond: 30,
			targetTicksPerSecond:  0,

			// 33.3 shared budget
			latencyFrame: time.Millisecond * 12,
			latencyTick:  time.Millisecond * 0,
		},
	}
}

func TestTraceExecutor(t *testing.T) {
	for _, variant := range testTraceVariants() {
		ctx, cancel := context.WithTimeout(context.Background(), variant.testDuration)

		testExecutor := NewExecutor(
			WithTargetFPS(variant.targetFramesPerSecond),
			WithTargetTPS(variant.targetTicksPerSecond),
			WithTask(
				NewDefaultTaskGarbageCollect(),
			),
		)

		collectedStats := make([]Stats, 0)
		measures := make([]testMeasure, 0)

		fnStats := func(s Stats) {
			collectedStats = append(collectedStats, s)
			measures = append(measures, waitMeasure(time.Millisecond*0, testTraceBlockSync))
		}

		fnFrame := func() error {
			measures = append(measures, waitMeasure(variant.latencyFrame, testTraceBlockFrame))
			return nil
		}

		fnTick := func() error {
			measures = append(measures, waitMeasure(variant.latencyTick, testTraceBlockTick))
			return nil
		}

		err := testExecutor.Execute(ctx, fnFrame, fnTick, fnStats)
		assert.NoError(t, err)

		cancel()

		testOutput(t, testExecutor, variant, measures, collectedStats)
	}
}

func testOutput(t *testing.T, e *Executor, variant testTraceVariant, measures []testMeasure, stats []Stats) {
	// colors
	const colBack = "#fff"
	const colText = "#001"
	const colTimeline = "#000"
	const colTimelineStrokeSecond = "#111"
	const colTimelineStrokeHalf = "#333"
	const colTimelineStroke100ms = "#555"
	const colTimelineStrokeBudget = "#999"
	const colBlockThrottle = "#777"
	const colBlockFrame = "#e40"
	const colBlockTick = "#0f3"
	const colBlockTask = "#02e"

	// const
	const widthPxPerSecond = float64(2000)
	const widthPxPerMs = widthPxPerSecond / 1000
	const sampleHeight = float64(50)
	const mainPaddingX = float64(20)
	const mainPaddingY = float64(40)
	const timeLineMargin = float64(4)
	const infoHeight = float64(15)

	// calculate graph size
	lastStat := stats[len(stats)-1]
	timeLineDurationMs := float64(lastStat.Execute.Duration.Milliseconds())
	timelineWidth := timeLineDurationMs * widthPxPerMs
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
	infoText := fmt.Sprintf("Frame: { lat:%dms, target: %d/s }  Tick: { lat:%dms, target: %d/s }",
		variant.latencyFrame.Milliseconds(),
		variant.targetFramesPerSecond,
		variant.latencyTick.Milliseconds(),
		variant.targetTicksPerSecond,
	)
	dc.DrawStringAnchored(infoText, mainPaddingX, 15, 0, 0)

	// timeline
	dc.SetHexColor(colTimeline)
	dc.DrawLine(mainPaddingX, timelineY, mainPaddingX+timelineWidth, timelineY)
	dc.Stroke()

	// timeline strokes
	drawStroke := func(interval time.Duration, color string, halfHeight float64, withText bool) {
		curTime := time.Millisecond * 0
		for x := mainPaddingX; x <= timelineWidth; x += float64(interval.Milliseconds()) * widthPxPerMs {
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
	drawStroke(lastStat.FrameTimeLimit, colTimelineStrokeBudget, 1, false)

	// blocks

	drawBlocks := func(samples []testMeasure, color string) {
		for _, sample := range samples {
			relativeStartAt := sample.startAt.Sub(lastStat.Execute.StartAt)
			x := mainPaddingX + (float64(relativeStartAt.Milliseconds()) * widthPxPerMs)
			width := float64(sample.endAt.Sub(sample.startAt).Milliseconds()) * widthPxPerMs

			dc.SetHexColor(color)
			dc.DrawRectangle(x, timelineY-timeLineMargin-sampleHeight, width, sampleHeight)
			dc.Fill()
		}
	}

	tasks := transformToMeasure(stats, func(s Stats) *testMeasure {
		return &testMeasure{
			bType:   testTraceBlockTask,
			startAt: s.Tasks.StartAt,
			endAt:   s.Tasks.StartAt.Add(s.Tasks.Duration),
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
