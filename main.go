package main

import (
	"flag"
	"fmt"
	"image"
	"log"
	"os"
	"time"

	"gocv.io/x/gocv"

	"github.com/thalesmeier/go-tracker/export"
	"github.com/thalesmeier/go-tracker/gui"
	"github.com/thalesmeier/go-tracker/tracker"
	"github.com/thalesmeier/go-tracker/video"
)

func main() {
	videoPath := flag.String("video", "", "Path to MP4 video file (required)")
	outputPath := flag.String("output", "tracking.csv", "Output CSV file path")
	templateSize := flag.Int("template-size", 15, "Template half-size in pixels")
	searchMargin := flag.Int("search-margin", 40, "Search margin in pixels")
	confidence := flag.Float64("confidence", 0.6, "Min confidence threshold (0-1)")
	startFrame := flag.Int("start-frame", 0, "Start tracking from this frame")
	showAxes := flag.Bool("axes", false, "Display X/Y axes through the tracking point")
	flag.Parse()

	if *videoPath == "" {
		fmt.Fprintf(os.Stderr, "Usage: go-tracker -video <path.mp4> [options]\n")
		flag.PrintDefaults()
		os.Exit(1)
	}

	reader, err := video.Open(*videoPath)
	if err != nil {
		log.Fatalf("Failed to open video: %v", err)
	}
	defer reader.Close()

	info := reader.Info()
	fmt.Printf("Video: %dx%d @ %.1f FPS, %d frames\n",
		info.Width, info.Height, info.FPS, info.FrameCount)

	if *startFrame > 0 {
		reader.Seek(*startFrame)
	}

	win := gui.New("GoTracker")
	defer win.Close()

	frame := gocv.NewMat()
	defer frame.Close()

	if !reader.Read(&frame) || frame.Empty() {
		log.Fatalf("Failed to read first frame")
	}

	fmt.Println("Click on the point to track, then tracking begins.")
	clickPt, _ := win.WaitClick(frame, "Click the point to track", nil)
	fmt.Printf("Selected point: (%d, %d)\n", clickPt.X, clickPt.Y)

	cfg := tracker.Config{
		TemplateSize:        *templateSize,
		SearchMargin:        *searchMargin,
		ConfidenceThreshold: *confidence,
		AdaptiveSearch:      true,
		MaxSearchMargin:     120,
	}

	t := tracker.New(cfg, info.FPS)
	defer t.Close()
	t.Initialize(frame, clickPt.X, clickPt.Y)

	frameNum := *startFrame
	stopped := false

	var totalDecode, totalTrack, totalDisplay time.Duration
	var framesProcessed int

	fmt.Println("Tracking... ESC=stop, Space=realign")
	for !stopped {
		frameNum++

		decodeStart := time.Now()
		if !reader.Read(&frame) || frame.Empty() {
			break
		}
		totalDecode += time.Since(decodeStart)

		trackStart := time.Now()
		state, tp := t.ProcessFrame(frame, frameNum)
		totalTrack += time.Since(trackStart)

		if state == tracker.StatePausedForRealignment {
			fmt.Printf("Lost track at frame %d. Click to realign.\n", frameNum)
			realign(win, t, frame, *showAxes)
			// Re-process this frame with new template
			state, tp = t.ProcessFrame(frame, frameNum)
		}

		if state == tracker.StateDone {
			break
		}

		framesProcessed++

		// Build overlay for display
		displayStart := time.Now()
		overlay := buildOverlay(t, tp, cfg, frameNum, info.FrameCount)
		overlay.ShowAxes = *showAxes
		key := win.ShowFrame(frame, overlay, 1)
		totalDisplay += time.Since(displayStart)

		switch {
		case key == 27: // ESC
			fmt.Println("Stopped by user.")
			stopped = true
		case key == 32 || key == 'p' || key == 'P': // Space or P
			fmt.Println("Manual realignment requested.")
			realign(win, t, frame, *showAxes)
		}
	}

	// Performance metrics
	if framesProcessed > 0 {
		totalPipeline := totalDecode + totalTrack + totalDisplay
		avgDecode := totalDecode / time.Duration(framesProcessed)
		avgTrack := totalTrack / time.Duration(framesProcessed)
		avgDisplay := totalDisplay / time.Duration(framesProcessed)
		avgTotal := totalPipeline / time.Duration(framesProcessed)
		trackingFPS := float64(framesProcessed) / totalPipeline.Seconds()

		fmt.Println("\n--- Performance ---")
		fmt.Printf("Frames processed: %d\n", framesProcessed)
		fmt.Printf("Tracking FPS:     %.1f\n", trackingFPS)
		fmt.Printf("Avg per frame:    %v (decode: %v, track: %v, display: %v)\n",
			avgTotal.Round(time.Microsecond),
			avgDecode.Round(time.Microsecond),
			avgTrack.Round(time.Microsecond),
			avgDisplay.Round(time.Microsecond))
	}

	points := t.Points()
	if len(points) == 0 {
		fmt.Println("No tracking data collected.")
		return
	}

	if err := export.WriteCSV(*outputPath, points); err != nil {
		log.Fatalf("Failed to write CSV: %v", err)
	}
	fmt.Printf("Exported %d points to %s\n", len(points), *outputPath)
}

func realign(win *gui.Window, t *tracker.Tracker, frame gocv.Mat, showAxes bool) {
	var pauseOverlay *gui.Overlay
	if showAxes {
		pauseOverlay = &gui.Overlay{
			TrackPos: t.LastPos(),
			ShowAxes: true,
			Status:   "PAUSED - Click to realign, Space to resume",
		}
	}
	pt, clicked := win.WaitClick(frame, "Click to realign, Space to resume", pauseOverlay)
	if !clicked {
		fmt.Println("Resumed from last position.")
		t.Resume()
		return
	}
	t.Realign(frame, pt.X, pt.Y)
	fmt.Printf("Realigned to (%d, %d)\n", pt.X, pt.Y)
}

func buildOverlay(t *tracker.Tracker, tp *export.TrackPoint, cfg tracker.Config, frameNum, totalFrames int) *gui.Overlay {
	pos := t.LastPos()
	halfT := cfg.TemplateSize
	margin := cfg.SearchMargin

	overlay := &gui.Overlay{
		TrackPos: pos,
		ROIRect: image.Rect(
			pos.X-halfT-margin, pos.Y-halfT-margin,
			pos.X+halfT+margin+1, pos.Y+halfT+margin+1,
		),
		Status: fmt.Sprintf("Frame %d/%d", frameNum, totalFrames),
	}

	if tp != nil {
		overlay.Confidence = tp.Confidence
	}

	return overlay
}
