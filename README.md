# Open Gallery
Open Gallery is a C++ library for generating dynamic, image-aware gallery layouts.

The library takes image metadata such as aspect ratio, focal region, importance, and prior placement, then generates and scores candidate layouts. The goal is to produce visually balanced, high-coverage gallery compositions while preserving important image content and avoiding unnecessary cropping.

## Project Goals
- Preserve image aspect ratios whenever possible.
- Use focal-region metadata to avoid cutting off important subjects.
- Generate varied layouts rather than relying only on fixed templates.
- Support multiple layout-generation strategies.
- Score candidate layouts using visual heuristics such as:
  - hierarchy
  - balance
  - coverage
  - spacing
  - alignment
  - focal-region preservation
  - image-to-tile compatibility
  - transition quality (Future TODO after mvp)

The core engine is independent from any particular UI, database, or web framewory and the memory and CPU usage are low enough for constrained hardware.

Architecture

The intended pipeline is:
1. Receive eligible image metadata.
2. Generate layout candidates using multiple generation strategies.
3. Eliminate structurally impossible candidates early.
4. Fit candidate images into candidate layouts.
5. Score complete image/layout compositions.
6. Refine the strongest candidates.
7. Return the best placement geometry.

The core library operates only on metadata and geometry.

Current Status

The project is early-stage. Prioritize correctness, clear architecture, and easy experimentation with layout algorithms over premature optimization or API stability.

Do not introduce additional dependencies or major architectural changes unless they provide a clear benefit to the layout engine.
