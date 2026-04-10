package export

import (
	"fmt"
	"image"
	"image/color"

	"gocv.io/x/gocv"
)

type VideoOptions struct {
	TrailLen int // number of trailing positions to draw (0 = all)
}

// WriteVideo re-reads the source video and writes an annotated copy with tracking overlay.
func WriteVideo(srcPath, dstPath string, points []TrackPoint, fps float64, startFrame int, opts VideoOptions) error {
	src, err := gocv.VideoCaptureFile(srcPath)
	if err != nil {
		return fmt.Errorf("export: failed to open source video: %w", err)
	}
	defer func() { _ = src.Close() }()

	w := int(src.Get(gocv.VideoCaptureFrameWidth))
	h := int(src.Get(gocv.VideoCaptureFrameHeight))

	writer, err := gocv.VideoWriterFile(dstPath, "avc1", fps, w, h, true)
	if err != nil {
		return fmt.Errorf("export: failed to create video writer: %w", err)
	}
	defer func() { _ = writer.Close() }()

	// Build a map from frame number to point index for quick lookup
	// Points are stored with Time = frameNum/fps, so frameNum = Time*fps
	pointByFrame := make(map[int]int, len(points))
	for i, p := range points {
		fn := int(p.Time*fps + 0.5)
		pointByFrame[fn] = i
	}

	frame := gocv.NewMat()
	defer func() { _ = frame.Close() }()

	green := color.RGBA{0, 255, 0, 0}
	frameNum := 0

	for src.Read(&frame) && !frame.Empty() {

		idx, hasPoint := pointByFrame[frameNum]

		if hasPoint {
			p := points[idx]
			pt := image.Pt(p.X, p.Y)

			// Draw trail
			trailStart := 0
			if opts.TrailLen > 0 && idx-opts.TrailLen > 0 {
				trailStart = idx - opts.TrailLen
			}
			if idx > trailStart {
				n := idx - trailStart
				for j := trailStart + 1; j <= idx; j++ {
					alpha := float64(j-trailStart) / float64(n)
					c := color.RGBA{
						R: uint8(alpha * 255),
						G: uint8((1 - alpha) * 200),
						A: 0,
					}
					prev := image.Pt(points[j-1].X, points[j-1].Y)
					cur := image.Pt(points[j].X, points[j].Y)
					_ = gocv.Line(&frame, prev, cur, c, 1)
				}
			}

			// Draw crosshair
			_ = gocv.Line(&frame, image.Pt(pt.X-10, pt.Y), image.Pt(pt.X+10, pt.Y), green, 2)
			_ = gocv.Line(&frame, image.Pt(pt.X, pt.Y-10), image.Pt(pt.X, pt.Y+10), green, 2)
		}

		if err := writer.Write(frame); err != nil {
			return fmt.Errorf("export: failed to write frame %d: %w", frameNum, err)
		}
		frameNum++
	}

	return nil
}
