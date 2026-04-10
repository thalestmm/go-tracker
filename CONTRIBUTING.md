# Contributing to GoTracker

Thanks for your interest in contributing! GoTracker is a lean physics tool — we value simplicity, performance, and practicality over feature count.

## Getting Started

### Prerequisites

- **Go** 1.25+: https://go.dev/dl/
- **OpenCV 4.x** with development headers
- **pkg-config**
- **just** (task runner): `brew install just` or https://github.com/casey/just
- **golangci-lint**: https://golangci-lint.run/welcome/install/

### Setup

```bash
git clone https://github.com/thalestmm/go-tracker.git
cd go-tracker
just build
just test
```

## Development Workflow

### Branch naming

- `feature/description` — new features
- `fix/description` — bug fixes
- `docs/description` — documentation changes
- `refactor/description` — code improvements without behavior change

### Commit style

Use [conventional commits](https://www.conventionalcommits.org/):

```
feat: add multi-point tracking
fix: clamp ROI to frame bounds near edges
docs: update README with calibration examples
refactor: extract overlay drawing into helper
```

### Before submitting a PR

```bash
just check    # runs fmt + lint + test
```

All three must pass. Your PR should include:

- A clear description of what changed and why
- Tests for new functionality (where feasible)
- No performance regression in turbo mode (check the metrics output)

## Code Style

- Run `golangci-lint fmt` to format code
- Keep functions small and focused
- No unnecessary abstractions — three similar lines beats a premature helper
- **Always close GoCV Mats.** Every `gocv.Mat` that is allocated must be closed. Use `defer mat.Close()` or explicit cleanup. Leaking Mats leaks C++ memory.
- Region views from `mat.Region()` should also be closed after use
- Reuse Mats where possible (allocate once, reuse across frames)

## Architecture

```
go-tracker/
├── main.go          # CLI flags, main tracking loop, orchestration
├── config/          # TOML config file loading
├── tracker/         # Core tracking: state machine, template matching, ROI
├── gui/             # OpenCV Highgui: windows, overlays, graphs
├── video/           # Video I/O: reading, seeking, metadata
└── export/          # Output: CSV writing, video export, derivatives
```

- **`tracker/`** is the algorithmic core — template matching, ROI management, adaptive search
- **`gui/`** handles all display — keep tracking logic out of here
- **`export/`** handles all output — CSV, video, scale computation
- **`main.go`** wires everything together — flag parsing, main loop, pause handling

## Testing

### Running tests

```bash
just test              # all tests
go test -v ./config/... ./export/...   # pure Go tests (no OpenCV needed)
go test -v ./tracker/...               # needs OpenCV installed
```

### What's testable

| Category | Approach | Example |
|----------|----------|---------|
| Pure computation | Standard unit tests | config parsing, CSV derivatives, ROI clamping, moving average |
| GoCV Mat operations | Synthetic Mats (headless) | template matching, tracker state machine |
| File I/O | Temp files via `t.TempDir()` | CSV writing, config loading |
| GUI / display | Manual testing only | window rendering, mouse interaction |

When adding new features:
- Extract computational logic into testable functions
- Keep display code thin — delegate to helpers that can be tested with synthetic Mats
- GUI-interactive code can't be unit tested; document manual test steps in your PR instead

### Writing tests with GoCV

```go
// Create a synthetic grayscale frame
frame := gocv.NewMatWithSize(100, 100, gocv.MatTypeCV8UC1)
defer frame.Close()

// Draw something on it
gocv.Rectangle(&frame, image.Rect(30, 30, 50, 50), color.RGBA{255, 255, 255, 0}, -1)
```

GoCV Mat creation and operations work without a display server. Only `IMShow`, `WaitKey`, and window functions require X11/Wayland.

## What NOT to Add

We intentionally keep the scope narrow. Please don't propose:

- Lens distortion correction
- ML-based object detection/tracking
- Real-time camera input
- Plugin systems
- Database storage
- Web UI

See [docs/roadmap.md](docs/roadmap.md) for planned features and the full anti-features list.

## Questions?

Open an issue on GitHub. For feature proposals, describe the physics use case first — we prioritize features that help students in lab settings.
