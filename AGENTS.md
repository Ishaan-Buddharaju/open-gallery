# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project overview

Open Gallery is a C++ library that takes a set of images and their subject/focal regions and computes the optimal layout for a rectangular gallery wall display. It generates candidate layouts with multiple generators, then scores each against a set of 6 heuristics of a "good layout" to pick the best one.

Planned/future work (not yet implemented): a transition-state heuristic, a vision service for automatically finding subjects, support for non-rectangular layouts.

## Current state of the repository

This is an early-stage project — there is no build system, test runner, or linter configured yet:

- `include/open_gallery/` — public headers (`.hh`). Currently contains `types.hh`, defining the core data types: `Point`, `FocalRegion`, `ImageMeta`.
- `src/` — empty, intended for implementation (`.cc`) files.
- `tests/` — empty, intended for tests.

There is no `CMakeLists.txt`, `Makefile`, or other build configuration yet. When adding one, check with the user for their preferred toolchain/build system rather than assuming.

## Code conventions

- C++ source files use `.cpp`.
- C++ headers use `.hh`.
- Types, structs, and classes use `PascalCase`.
- Functions and variables use `camelCase`.
- Public headers live under `include/open_gallery/`.
- Implementation details should stay under `src/`.
- Prefer simple data structures and explicit algorithms over unnecessary abstraction.
- Keep public APIs small and platform-independent.

## Core data model (`include/open_gallery/types.hh`)

- `Point` — a simple `{x, y}` float pair.
- `FocalRegion` — the union of all subject/focal regions within an image, defined by a top-left and bottom-right `Point`. Its constructor takes pixel coordinates plus the image's pixel width/height and immediately normalizes the corner points to the `[0, 1]` range (dividing by image dimensions), so that regions can be reasoned about independent of pixel resolution. The constructor throws `std::invalid_argument` for invalid image dimensions or out-of-bounds/inverted region coordinates.
- `ImageMeta` — metadata for a single image: `id`, pixel `widthPx`/`heightPx`, `aspectRatio`, and an optional `subject` (`FocalRegion*`, the focal region for that image).

When extending this model, keep normalization of coordinates (image-space vs. unit `[0,1]` space) consistent with how `FocalRegion` already does it.
