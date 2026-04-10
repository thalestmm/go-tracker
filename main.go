package main

import (
	"bufio"
	"flag"
	"fmt"
	"image"
	"log"
	"os"
	"strconv"
	"strings"
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
	turbo := flag.Bool("turbo", false, "Skip rendering during tracking for maximum speed (display only on pause)")
	exportConf := flag.Bool("export-confidence", false, "Include confidence column in CSV output")
	calibrate := flag.Bool("calibrate", false, "Calibrate pixel-to-real-world scale before tracking")
	scaleUnit := flag.String("unit", "m", "Unit label for calibrated output (e.g. m, cm, mm)")
	showGraph := flag.Bool("graph", false, "Show real-time X(t) and Y(t) graph window")
	trailLen := flag.Int("trail", 0, "Draw trajectory trail of last N positions (0=off)")
	exportVideo := flag.String("export-video", "", "Export annotated video to this path (e.g. output.mp4)")
	derivatives := flag.Bool("derivatives", false, "Include vx, vy, ax, ay columns in CSV output")
	startTime := flag.Float64("start-time", 0, "Start tracking from this time in seconds (overrides -start-frame)")
	flag.Parse()

	// --- 1.1: Input validation ---
	if *videoPath == "" {
		fmt.Fprintf(os.Stderr, "Usage: go-tracker -video <path.mp4> [options]\n")
		flag.PrintDefaults()
		os.Exit(1)
	}
	if *confidence < 0.0 || *confidence > 1.0 {
		log.Fatalf("Confidence must be between 0.0 and 1.0, got %.2f", *confidence)
	}
	if *templateSize <= 0 {
		log.Fatalf("Template size must be positive, got %d", *templateSize)
	}
	if *searchMargin <= 0 {
		log.Fatalf("Search margin must be positive, got %d", *searchMargin)
	}
	if *startFrame < 0 {
		log.Fatalf("Start frame must be non-negative, got %d", *startFrame)
	}

	reader, err := video.Open(*videoPath)
	if err != nil {
		log.Fatalf("Failed to open video: %v", err)
	}
	defer reader.Close()

	info := reader.Info()
	fmt.Printf("Video: %dx%d @ %.1f FPS, %d frames\n",
		info.Width, info.Height, info.FPS, info.FrameCount)

	// Convert start-time to start-frame if specified
	if *startTime > 0 {
		*startFrame = int(*startTime * info.FPS)
		fmt.Printf("Starting at %.2fs (frame %d)\n", *startTime, *startFrame)
	}

	// --- 1.4: Validate start frame ---
	if info.FrameCount > 0 && *startFrame >= info.FrameCount {
		log.Fatalf("Start frame %d exceeds video length (%d frames)", *startFrame, info.FrameCount)
	}

	// --- 1.4: Validate template fits in video ---
	minDim := info.Width
	if info.Height < minDim {
		minDim = info.Height
	}
	roiSize := 2*(*templateSize+*searchMargin) + 1
	if roiSize > minDim {
		log.Fatalf("Template + margin (%d px) exceeds smallest video dimension (%d px). Reduce -template-size or -search-margin.", roiSize, minDim)
	}

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

	// --- 2.1: Scale calibration ---
	var pixelsPerUnit float64
	if *calibrate {
		fmt.Println("Calibration: click two reference points with a known distance.")
		p1, p2 := win.WaitTwoClicks(frame,
			"Click first calibration point",
			"Click second calibration point")
		fmt.Printf("Calibration points: (%d,%d) -> (%d,%d)\n", p1.X, p1.Y, p2.X, p2.Y)

		dist := readFloat("Enter the real-world distance between these points (in " + *scaleUnit + "): ")
		if dist <= 0 {
			log.Fatalf("Distance must be positive")
		}
		pixelsPerUnit = export.ComputeScale([2]int{p1.X, p1.Y}, [2]int{p2.X, p2.Y}, dist)
		fmt.Printf("Scale: %.2f pixels/%s\n", pixelsPerUnit, *scaleUnit)
	}

	// --- Point selection with zoom preview ---
	fmt.Println("Click on the point to track. A zoom inset will appear for confirmation.")
	clickPt, _ := win.WaitClickZoom(frame, "Click the point to track", *templateSize+5)

	// --- 1.4: Validate click is within frame ---
	if clickPt.X < 0 || clickPt.X >= info.Width || clickPt.Y < 0 || clickPt.Y >= info.Height {
		log.Fatalf("Selected point (%d, %d) is outside frame bounds", clickPt.X, clickPt.Y)
	}
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

	// Trail buffer for trajectory overlay
	var trailBuf []image.Point

	// Graph window for real-time plotting
	var graphWin *gui.GraphWindow
	var graphTimes []float64
	var graphXs, graphYs []int
	if *showGraph {
		graphWin = gui.NewGraphWindow("GoTracker - Graph", *derivatives)
		defer graphWin.Close()
	}

	if *turbo {
		fmt.Println("Tracking (turbo mode)... auto-pauses on lost track")
		win.ShowTurboLabel(frame)
		win.PollKey(1)
	} else {
		fmt.Println("Tracking... ESC=stop, Space=pause")
	}

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
			state, tp = t.ProcessFrame(frame, frameNum)
		}

		if state == tracker.StateDone {
			break
		}

		framesProcessed++

		// Update trail buffer
		if *trailLen > 0 && tp != nil {
			trailBuf = append(trailBuf, image.Pt(tp.X, tp.Y))
			if len(trailBuf) > *trailLen {
				trailBuf = trailBuf[len(trailBuf)-*trailLen:]
			}
		}

		// Update graph with new data point
		if graphWin != nil && tp != nil {
			graphTimes = append(graphTimes, tp.Time)
			graphXs = append(graphXs, tp.X)
			graphYs = append(graphYs, tp.Y)
			graphWin.Update(graphTimes, graphXs, graphYs)
		}

		if !*turbo {
			displayStart := time.Now()
			overlay := buildOverlay(t, tp, cfg, frameNum, info.FrameCount)
			overlay.ShowAxes = *showAxes
			overlay.Trail = trailBuf
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

	// --- 1.2 + 2.1: CSV export with options ---
	csvOpts := export.CSVOptions{
		IncludeConfidence: *exportConf,
		Scale:             pixelsPerUnit,
		ScaleUnit:         *scaleUnit,
		Derivatives:       *derivatives,
	}
	if err := export.WriteCSV(*outputPath, points, csvOpts); err != nil {
		log.Fatalf("Failed to write CSV: %v", err)
	}
	fmt.Printf("Exported %d points to %s\n", len(points), *outputPath)

	// --- Annotated video export ---
	if *exportVideo != "" {
		fmt.Printf("Exporting annotated video to %s...\n", *exportVideo)
		vidOpts := export.VideoOptions{TrailLen: *trailLen}
		if err := export.WriteVideo(*videoPath, *exportVideo, points, info.FPS, *startFrame, vidOpts); err != nil {
			log.Fatalf("Failed to export video: %v", err)
		}
		fmt.Printf("Video exported to %s\n", *exportVideo)
	}
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

func readFloat(prompt string) float64 {
	fmt.Print(prompt)
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	text := strings.TrimSpace(scanner.Text())
	val, err := strconv.ParseFloat(text, 64)
	if err != nil {
		log.Fatalf("Invalid number: %q", text)
	}
	return val
}
