package video

import (
	"fmt"
	"log"

	"gocv.io/x/gocv"
)

type VideoInfo struct {
	Path       string
	Width      int
	Height     int
	FPS        float64
	FrameCount int
}

type Reader struct {
	cap  *gocv.VideoCapture
	info VideoInfo
}

func Open(path string) (*Reader, error) {
	cap, err := gocv.VideoCaptureFile(path)
	if err != nil {
		return nil, fmt.Errorf("video: failed to open %s: %w", path, err)
	}

	fps := cap.Get(gocv.VideoCaptureFPS)
	if fps <= 0 {
		log.Printf("video: warning: invalid FPS (%.2f), defaulting to 30.0", fps)
		fps = 30.0
	}

	info := VideoInfo{
		Path:       path,
		Width:      int(cap.Get(gocv.VideoCaptureFrameWidth)),
		Height:     int(cap.Get(gocv.VideoCaptureFrameHeight)),
		FPS:        fps,
		FrameCount: int(cap.Get(gocv.VideoCaptureFrameCount)),
	}

	return &Reader{cap: cap, info: info}, nil
}

func (r *Reader) Read(dst *gocv.Mat) bool {
	return r.cap.Read(dst)
}

func (r *Reader) Seek(frame int) error {
	r.cap.Set(gocv.VideoCapturePosFrames, float64(frame))
	return nil
}

func (r *Reader) Info() VideoInfo {
	return r.info
}

func (r *Reader) Close() {
	r.cap.Close()
}
