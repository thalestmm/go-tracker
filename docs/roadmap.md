# GoTracker Roadmap

Future improvements planned for GoTracker. Each item is independent and can be implemented in any order. The guiding principle: stay lean, stay fast, stay practical for physics lab use.

---

## Wire up CUDA matcher

The CUDA template matching implementation exists (`tracker/matcher_cuda.go`) but is never called from the main code path — `NewMatcher()` always returns the CPU backend. Add runtime detection so that when built with the `cuda` tag and a GPU is available, it's used automatically. Include auto-fallback to CPU for search regions under 10,000 pixels where GPU overhead exceeds the benefit.

**Files:** `tracker/matcher.go`, `tracker/matcher_cuda.go`

---

## Multi-point tracking (`-points N`)

Many physics experiments involve two or more objects: collisions, coupled pendulums, center-of-mass studies. The user clicks N points on the first frame, each getting its own template. All points are tracked per frame with a single grayscale conversion and N independent template matches (~0.3ms each, so performance stays high). If any point is lost, tracking pauses and highlights which point needs realignment. CSV output uses one row per frame with columns `time, x1, y1, x2, y2, ...`.

**Files:** `tracker/tracker.go` (multi-state tracking), `gui/window.go` (multi-point selection and overlay), `export/csv.go` (wide-format output), `main.go`

---

## Progress output in turbo mode

In turbo mode there's no GUI feedback. Print a single-line progress indicator to the terminal every ~500ms using carriage return (`\r`) to avoid scroll spam: `Frame 2500/5329 (47%) — 1710 FPS`. Gives the user confidence that tracking is progressing without any performance cost.

**Files:** `main.go`

---

## Auto-detect good tracking points

Novice users sometimes click on featureless areas (uniform color, flat textures) that are hard to track reliably. Before accepting a click, compute a local "trackability" score around the selected point using OpenCV's corner response (`gocv.GoodFeaturesToTrack` or similar). If the score is low, show a non-blocking warning: "Low contrast area — tracking may drift. Click again or press Enter to confirm." The user can always override.

**Files:** `gui/window.go`, `tracker/roi.go`

---

## Undo realignment

After realigning, pressing `u` or `Ctrl+Z` reverts to the previous template and position — one level of undo. Just keep the old template in memory alongside the current one. Cheap in both memory and complexity, but saves time when a mis-click during realignment would otherwise require restarting.

**Files:** `tracker/tracker.go`, `main.go`

---

## Config file support

Read default flag values from `~/.go-tracker.toml` or a project-local `.go-tracker.toml`. Avoids long CLI flag strings for repeated use (common in lab settings where the same experiment is filmed multiple times). CLI flags always override config file values.

---

## Batch processing

Accept a directory of videos via `-batch <dir>`, process each sequentially with the same tracking parameters. Output one CSV per video with matching filenames. Useful for repetitive lab work where the same experiment is recorded across multiple trials.

---

## JSON/Parquet output (`-format json|parquet`)

Alternative output formats for students using pandas or polars. JSON is trivial to add; Parquet would need a third-party dependency. Only add if there's actual demand — CSV is universal.

---

## ROI preview before tracking

After clicking to select a tracking point, show the actual template patch and search region overlaid on the frame. User confirms with Enter or re-clicks. Prevents wasted tracking runs from bad selections, especially useful for beginners.

---

## What we intentionally won't add

- **Lens distortion correction** — use external calibration tools
- **Object detection / ML tracking** — breaks the lean principle, adds huge dependencies
- **Real-time camera input** — different use case, different architecture
- **Plugin system** — over-engineering for this scope
- **Database storage** — CSV is universal and sufficient
- **Web UI** — the OpenCV window works; a web UI adds server, JS, websockets
