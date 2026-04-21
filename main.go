package main

import (
	"bufio"
	"flag"
	"fmt"
	"image"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"gocv.io/x/gocv"

	"github.com/thalestmm/go-tracker/config"
	"github.com/thalestmm/go-tracker/export"
	"github.com/thalestmm/go-tracker/gui"
	"github.com/thalestmm/go-tracker/tracker"
	"github.com/thalestmm/go-tracker/video"
)

type runFlags struct {
	startFrame   int
	startTime    float64
	showAxes     bool
	turbo        bool
	exportConf   bool
	calibrate    bool
	scaleUnit    string
	showGraph    bool
	trailLen     int
	derivatives  bool
	smooth       int
}

type batchJob struct {
	videoPath       string
	outputPath      string
	exportVideoPath string
}

type batchFailure struct {
	path string
	err  error
}

var videoExts = map[string]bool{
	".mp4": true,
	".mov": true,
	".avi": true,
	".mkv": true,
}

func main() {
	// Config file flag is parsed first
	configPath := flag.String("config", "", "Path to config file (default: ./go-tracker.toml)")

	videoPath := flag.String("video", "", "Path to video file (.mp4, .mov, .avi, .mkv)")
	batchDir := flag.String("batch", "", "Process all videos in this directory sequentially (mutually exclusive with -video)")
	outputPath := flag.String("output", "", "Output CSV file path (ignored in -batch mode)")
	templateSize := flag.Int("template-size", 0, "Template half-size in pixels")
	searchMargin := flag.Int("search-margin", 0, "Search margin in pixels")
	confidence := flag.Float64("confidence", 0, "Min confidence threshold (0-1)")
	startFrame := flag.Int("start-frame", 0, "Start tracking from this frame")
	showAxes := flag.Bool("axes", false, "Display X/Y axes through the tracking point")
	turbo := flag.Bool("turbo", false, "Skip rendering during tracking for maximum speed (display only on pause)")
	exportConf := flag.Bool("export-confidence", false, "Include confidence column in CSV output")
	calibrate := flag.Bool("calibrate", false, "Calibrate pixel-to-real-world scale before tracking")
	scaleUnit := flag.String("unit", "", "Unit label for calibrated output (e.g. m, cm, mm)")
	showGraph := flag.Bool("graph", false, "Show real-time X(t) and Y(t) graph window")
	trailLen := flag.Int("trail", 0, "Draw trajectory trail of last N positions (0=off)")
	exportVideo := flag.String("export-video", "", "Export annotated video (path in single mode; any non-empty value enables per-video auto-naming in -batch mode)")
	derivatives := flag.Bool("derivatives", false, "Include vx, vy, ax, ay columns in CSV output")
	startTime := flag.Float64("start-time", 0, "Start tracking from this time in seconds (overrides -start-frame)")
	smooth := flag.Int("smooth", 0, "Smoothing window for graph display (0=off, e.g. 5 or 10). Does not affect CSV output.")
	flag.Parse()

	// Load config file: explicit path > ./go-tracker.toml > defaults
	cfgFile := *configPath
	if cfgFile == "" {
		cfgFile = "go-tracker.toml"
	}
	cfg, err := config.Load(cfgFile)
	if err != nil && *configPath != "" {
		// Only fatal if user explicitly passed a config path
		log.Fatalf("Failed to load config: %v", err)
	}

	// CLI flags override config values (only when explicitly set)
	if *outputPath == "" {
		*outputPath = cfg.Output
	}
	if *templateSize == 0 {
		*templateSize = cfg.TemplateSize
	}
	if *searchMargin == 0 {
		*searchMargin = cfg.SearchMargin
	}
	if *confidence == 0 {
		*confidence = cfg.Confidence
	}
	if *scaleUnit == "" {
		*scaleUnit = cfg.Unit
	}
	if !*showAxes && cfg.Axes {
		*showAxes = true
	}
	if !*turbo && cfg.Turbo {
		*turbo = true
	}
	if !*exportConf && cfg.ExportConfidence {
		*exportConf = true
	}
	if !*calibrate && cfg.Calibrate {
		*calibrate = true
	}
	if !*showGraph && cfg.Graph {
		*showGraph = true
	}
	if *trailLen == 0 && cfg.Trail > 0 {
		*trailLen = cfg.Trail
	}
	if *exportVideo == "" && cfg.ExportVideo != "" {
		*exportVideo = cfg.ExportVideo
	}
	if !*derivatives && cfg.Derivatives {
		*derivatives = true
	}
	if *smooth == 0 && cfg.Smooth > 0 {
		*smooth = cfg.Smooth
	}
	if *startFrame == 0 && cfg.StartFrame > 0 {
		*startFrame = cfg.StartFrame
	}
	if *startTime == 0 && cfg.StartTime > 0 {
		*startTime = cfg.StartTime
	}

	// --- Input validation ---
	if *videoPath == "" && *batchDir == "" {
		fmt.Fprintf(os.Stderr, "Usage: go-tracker (-video <path.mp4> | -batch <dir>) [options]\n")
		flag.PrintDefaults()
		os.Exit(1)
	}
	if *videoPath != "" && *batchDir != "" {
		log.Fatalf("-video and -batch are mutually exclusive")
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

	tcfg := tracker.Config{
		TemplateSize:        *templateSize,
		SearchMargin:        *searchMargin,
		ConfidenceThreshold: *confidence,
		AdaptiveSearch:      true,
		MaxSearchMargin:     120,
	}

	flags := runFlags{
		startFrame:  *startFrame,
		startTime:   *startTime,
		showAxes:    *showAxes,
		turbo:       *turbo,
		exportConf:  *exportConf,
		calibrate:   *calibrate,
		scaleUnit:   *scaleUnit,
		showGraph:   *showGraph,
		trailLen:    *trailLen,
		derivatives: *derivatives,
		smooth:      *smooth,
	}

	jobs, err := buildJobs(*batchDir, *videoPath, *outputPath, *exportVideo)
	if err != nil {
		log.Fatalf("%v", err)
	}
	if len(jobs) == 0 {
		log.Fatalf("No video files found in %s", *batchDir)
	}
	if *batchDir != "" && *outputPath != "" && *outputPath != cfg.Output {
		fmt.Fprintf(os.Stderr, "warning: -output is ignored in -batch mode; CSV names derived from video filenames\n")
	}

	win := gui.New("GoTracker")
	defer win.Close()

	var failures []batchFailure
	for i, j := range jobs {
		if len(jobs) > 1 {
			fmt.Printf("\n=== [%d/%d] %s ===\n", i+1, len(jobs), j.videoPath)
		}
		if err := runSingle(j, win, tcfg, flags); err != nil {
			log.Printf("FAILED %s: %v", j.videoPath, err)
			failures = append(failures, batchFailure{j.videoPath, err})
			continue
		}
	}

	if *batchDir != "" {
		succeeded := len(jobs) - len(failures)
		fmt.Printf("\n=== Batch done: %d succeeded, %d failed ===\n", succeeded, len(failures))
		for _, f := range failures {
			fmt.Printf("  FAIL %s: %v\n", f.path, f.err)
		}
		if len(failures) > 0 {
			os.Exit(1)
		}
	} else if len(failures) > 0 {
		// Single-video mode: propagate the error as a fatal exit.
		log.Fatalf("%v", failures[0].err)
	}
}

// buildJobs expands the CLI inputs into a list of per-video jobs.
func buildJobs(batchDir, videoPath, outputPath, exportVideo string) ([]batchJob, error) {
	if batchDir == "" {
		return []batchJob{{
			videoPath:       videoPath,
			outputPath:      outputPath,
			exportVideoPath: exportVideo,
		}}, nil
	}

	entries, err := os.ReadDir(batchDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read batch dir: %w", err)
	}

	var jobs []batchJob
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		ext := strings.ToLower(filepath.Ext(name))
		if !videoExts[ext] {
			continue
		}
		base := strings.TrimSuffix(name, filepath.Ext(name))
		in := filepath.Join(batchDir, name)
		out := filepath.Join(batchDir, base+".csv")
		var ev string
		if exportVideo != "" {
			ev = filepath.Join(batchDir, base+"_annotated.mp4")
		}
		jobs = append(jobs, batchJob{videoPath: in, outputPath: out, exportVideoPath: ev})
	}
	sort.Slice(jobs, func(i, j int) bool { return jobs[i].videoPath < jobs[j].videoPath })
	return jobs, nil
}

// runSingle runs the full tracking pipeline against one video. Errors are
// returned rather than fatal so batch mode can continue past failures.
func runSingle(job batchJob, win *gui.Window, tcfg tracker.Config, flags runFlags) error {
	reader, err := video.Open(job.videoPath)
	if err != nil {
		return fmt.Errorf("failed to open video: %w", err)
	}
	defer reader.Close()

	info := reader.Info()
	fmt.Printf("Video: %dx%d @ %.1f FPS, %d frames\n",
		info.Width, info.Height, info.FPS, info.FrameCount)

	startFrame := flags.startFrame
	if flags.startTime > 0 {
		startFrame = int(flags.startTime * info.FPS)
		fmt.Printf("Starting at %.2fs (frame %d)\n", flags.startTime, startFrame)
	}

	if info.FrameCount > 0 && startFrame >= info.FrameCount {
		return fmt.Errorf("start frame %d exceeds video length (%d frames)", startFrame, info.FrameCount)
	}

	minDim := min(info.Width, info.Height)
	roiSize := 2*(tcfg.TemplateSize+tcfg.SearchMargin) + 1
	if roiSize > minDim {
		return fmt.Errorf("template + margin (%d px) exceeds smallest video dimension (%d px). Reduce -template-size or -search-margin", roiSize, minDim)
	}

	if startFrame > 0 {
		_ = reader.Seek(startFrame)
	}

	frame := gocv.NewMat()
	defer func() { _ = frame.Close() }()

	if !reader.Read(&frame) || frame.Empty() {
		return fmt.Errorf("failed to read first frame")
	}

	var pixelsPerUnit float64
	if flags.calibrate {
		fmt.Println("Calibration: click two reference points with a known distance.")
		p1, p2 := win.WaitTwoClicks(frame,
			"Click first calibration point",
			"Click second calibration point")
		fmt.Printf("Calibration points: (%d,%d) -> (%d,%d)\n", p1.X, p1.Y, p2.X, p2.Y)

		dist := readFloat("Enter the real-world distance between these points (in " + flags.scaleUnit + "): ")
		if dist <= 0 {
			return fmt.Errorf("distance must be positive")
		}
		pixelsPerUnit = export.ComputeScale([2]int{p1.X, p1.Y}, [2]int{p2.X, p2.Y}, dist)
		fmt.Printf("Scale: %.2f pixels/%s\n", pixelsPerUnit, flags.scaleUnit)
	}

	fmt.Println("Click on the point to track. A zoom inset will appear for confirmation.")
	clickPt, _ := win.WaitClickZoom(frame, "Click the point to track", tcfg.TemplateSize+5)

	if clickPt.X < 0 || clickPt.X >= info.Width || clickPt.Y < 0 || clickPt.Y >= info.Height {
		return fmt.Errorf("selected point (%d, %d) is outside frame bounds", clickPt.X, clickPt.Y)
	}
	fmt.Printf("Selected point: (%d, %d)\n", clickPt.X, clickPt.Y)

	t := tracker.New(tcfg, info.FPS)
	defer t.Close()
	t.Initialize(frame, clickPt.X, clickPt.Y)

	frameNum := startFrame
	stopped := false

	var totalDecode, totalTrack, totalDisplay time.Duration
	var framesProcessed int

	var trailBuf []image.Point

	var graphWin *gui.GraphWindow
	var graphTimes []float64
	var graphXs, graphYs []int
	if flags.showGraph {
		graphWin = gui.NewGraphWindow("GoTracker - Graph", flags.derivatives, flags.smooth)
		defer graphWin.Close()
	}

	if flags.turbo {
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
			frameNum = pauseLoop(win, t, reader, &frame, frameNum, flags.showAxes)
			state, tp = t.ProcessFrame(frame, frameNum)
		}

		if state == tracker.StateDone {
			break
		}

		framesProcessed++

		if flags.trailLen > 0 && tp != nil {
			trailBuf = append(trailBuf, image.Pt(tp.X, tp.Y))
			if len(trailBuf) > flags.trailLen {
				trailBuf = trailBuf[len(trailBuf)-flags.trailLen:]
			}
		}

		if graphWin != nil && tp != nil {
			graphTimes = append(graphTimes, tp.Time)
			graphXs = append(graphXs, tp.X)
			graphYs = append(graphYs, tp.Y)
			graphWin.Update(graphTimes, graphXs, graphYs)
		}

		if !flags.turbo {
			displayStart := time.Now()
			overlay := buildOverlay(t, tp, tcfg, frameNum, info.FrameCount)
			overlay.ShowAxes = flags.showAxes
			overlay.Trail = trailBuf
			key := win.ShowFrame(frame, overlay, 1)
			totalDisplay += time.Since(displayStart)

			switch key {
			case 27: // ESC
				fmt.Println("Stopped by user.")
				stopped = true
			case 32, 'p', 'P': // Space or P
				fmt.Println("Paused.")
				frameNum = pauseLoop(win, t, reader, &frame, frameNum, flags.showAxes)
			}
		}
	}

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
		return nil
	}

	csvOpts := export.CSVOptions{
		IncludeConfidence: flags.exportConf,
		Scale:             pixelsPerUnit,
		ScaleUnit:         flags.scaleUnit,
		Derivatives:       flags.derivatives,
	}
	if err := export.WriteCSV(job.outputPath, points, csvOpts); err != nil {
		return fmt.Errorf("failed to write CSV: %w", err)
	}
	fmt.Printf("Exported %d points to %s\n", len(points), job.outputPath)

	if job.exportVideoPath != "" {
		fmt.Printf("Exporting annotated video to %s...\n", job.exportVideoPath)
		vidOpts := export.VideoOptions{TrailLen: flags.trailLen}
		if err := export.WriteVideo(job.videoPath, job.exportVideoPath, points, info.FPS, startFrame, vidOpts); err != nil {
			return fmt.Errorf("failed to export video: %w", err)
		}
		fmt.Printf("Video exported to %s\n", job.exportVideoPath)
	}

	return nil
}

// pauseLoop handles the pause state: user can resume, realign, or step frames.
// Returns the (possibly updated) frame number.
func pauseLoop(win *gui.Window, t *tracker.Tracker, reader *video.Reader, frame *gocv.Mat, frameNum int, showAxes bool) int {
	for {
		overlay := &gui.Overlay{
			TrackPos: t.LastPos(),
			ShowAxes: showAxes,
			Status:   fmt.Sprintf("PAUSED - Frame %d", frameNum),
		}

		result := win.WaitPause(*frame, overlay)

		switch result.Action {
		case gui.PauseResume:
			fmt.Println("Resumed from last position.")
			t.Resume()
			return frameNum
		case gui.PauseClick:
			t.Realign(*frame, result.ClickPt.X, result.ClickPt.Y)
			fmt.Printf("Realigned to (%d, %d)\n", result.ClickPt.X, result.ClickPt.Y)
			return frameNum
		case gui.PauseStepFwd:
			if reader.Read(frame) && !frame.Empty() {
				frameNum++
				// Track the new frame so the crosshair follows the object
				t.Resume()
				_, tp := t.ProcessFrame(*frame, frameNum)
				if tp != nil {
					fmt.Printf("Step frame %d -> (%d, %d) conf=%.2f\n", frameNum, tp.X, tp.Y, tp.Confidence)
				} else {
					fmt.Printf("Step frame %d (lost track)\n", frameNum)
				}
			}
		case gui.PauseStepBack:
			if frameNum > 0 {
				frameNum--
				_ = reader.Seek(frameNum)
				_ = reader.Read(frame)
				fmt.Printf("Stepped back to frame %d\n", frameNum)
			}
		}
	}
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
