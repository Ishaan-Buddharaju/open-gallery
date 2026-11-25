# AGENTS.md

This file provides guidance to AI coding agents working in this repository.

## Project overview

Open Gallery is a C++17 library for generating image-aware layouts for a rectangular gallery wall display. The intended pipeline generates candidates using multiple strategies, rejects structurally impossible candidates, fits images, scores compositions, refines the strongest candidates, and returns the best placement geometry. The core library operates only on metadata and geometry.

Currently implemented: the core data model and a uniform grid layout generator. Candidate scoring, image fitting, refinement, and additional generators are not yet implemented. Planned scoring criteria include hierarchy, balance, coverage, spacing, alignment, focal-region preservation, and image-to-tile compatibility. Transition quality, automatic subject detection, and non-rectangular layouts are future work.

## Current state of the repository

This is an early-stage project with a small CMake build and assertion-based tests registered with CTest. No linter is configured:

- `include/open_gallery/` — public headers (`.hh`). Currently contains the core data model in `types.hh` and generator headers under `generators/`.
- `src/` — implementation files (`.cpp`). Current generator implementations live under `src/generators/`.
- `src/generators/grid.cpp` — uniform grid layout generator.
- `include/open_gallery/generators/grid.hh` — public declaration for `generateGridLayout`.
- `src/generators/generator_methods.md` — design notes for planned generation strategies, including spaced grids; these describe proposed behavior rather than the current grid implementation.
- `tests/test_open_gallery.cpp` — tests for region normalization and validation, image aspect-ratio updates, and grid generation.
- `tests/test_grid.cpp` — dedicated generator tests for expected cell geometry, extreme display shapes, and coverage, bounds, non-overlap, and image assignment across varied image counts and display sizes. Built as `open_gallery_grid_tests` and registered with CTest.
- `CMakeLists.txt` — builds the `open_gallery` library and, when `BUILD_TESTING` is enabled, the `open_gallery_tests` executable registered with CTest.

When adding new generators, add the public header under `include/open_gallery/generators/`, the implementation under `src/generators/`, and update `CMakeLists.txt` so the source is part of the `open_gallery` target.

## Build and test

From the repository root:

```sh
cmake -S . -B build -DCMAKE_BUILD_TYPE=Debug -DBUILD_TESTING=ON
cmake --build build
ctest --test-dir build --output-on-failure
```

Tests use standard `assert`, so use a configuration without `NDEBUG` (such as Debug) to keep checks active. Add relevant correctness tests when changing layout algorithms or validation behavior.

## Code conventions

- C++ source files use `.cpp`.
- C++ headers use `.hh`.
- Types, structs, and classes use `PascalCase`.
- Functions and variables use `camelCase`.
- Public headers live under `include/open_gallery/`.
- Implementation details should stay under `src/`.
- Prefer simple data structures and explicit algorithms over unnecessary abstraction.
- Keep public APIs small and platform-independent.
- Prioritize correctness, clear architecture, and easy experimentation over premature optimization or API stability.
- Avoid new dependencies or major architectural changes unless they provide a clear benefit to the layout engine.

## Core data model (`include/open_gallery/types.hh`)

- `Point` — a simple `{x, y}` float pair.
- `Region` — a normalized rectangle, used both for image subject/focal regions and for image placement rectangles inside a `Layout`. Its constructor takes pixel coordinates plus a width/height and normalizes the corner points to `[0, 1]`. It throws `std::invalid_argument` for invalid dimensions or out-of-bounds/inverted coordinates.
- `ImageMeta` — metadata for a single image: pixel `widthPx`/`heightPx`, derived `aspectRatio`, and an optional non-owning `subject` pointer (`Region*`).
- `Placement` — pairs a placement `Region` with a non-owning `ImageMeta` pointer.
- `Layout` — one candidate gallery layout with display dimensions, derived aspect ratio, and a vector of placements.

When extending this model, keep normalization of coordinates consistent with `Region`: pass pixel-space rectangles into the constructor along with the coordinate space dimensions, and store/query normalized `[0, 1]` points afterward.

## Existing generator pattern

- Public generator functions currently return a `Layout`.
- `generateGridLayout(const std::vector<ImageMeta>& images, uint16_t displayWidthPx, uint16_t displayHeightPx)` is the reference implementation.
- The current grid chooses a column count based on image count and display aspect ratio. Each row fills the display width; an incomplete final row uses wider cells. It does not preserve source-image aspect ratios or compute focal-aware crops.
- Generators should validate display dimensions, return an empty `Layout` for an empty image list, and create placement `Region`s in display pixel coordinates.
- The generator assigns non-owning pointers to the `ImageMeta` elements in the input vector; callers must keep the image vector alive while using the returned `Layout`.
