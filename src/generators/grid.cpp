#include <vector>
#include <cmath>
#include <algorithm>

#include <open_gallery/types.hh>
#include <open_gallery/generators/grid.hh>

// Layout generateSpacedGrid(std::vector<ImageMeta> images) {
//     uint64_t imageCount = images.size();

// }

/*
 * generateGridLayout(images, displayWidthPx, displayHeightPx)
 *
 * Splits the display into a uniform grid of cells, one per image, and returns a Layout
 * pairing each image with the Region of its cell. Cells are sized to fully cover their
 * row (crop-to-cell) rather than being shrunk/letterboxed to preserve each image's own
 * aspect ratio; deciding which part of the source image to crop (e.g. protecting
 * ImageMeta::subject()) is left to rendering, not this generator.
 */
Layout generateGridLayout(const std::vector<ImageMeta>& images, uint16_t displayWidthPx, uint16_t displayHeightPx) {
    if (displayWidthPx <= 0 || displayHeightPx <= 0) {
        throw std::invalid_argument("Invalid display dimensions");
    }

    Layout layout(displayWidthPx, displayHeightPx);
    float displayAspect = layout.aspectRatio();

    size_t n = images.size();
    if (n == 0) {
        return layout;
    }

    /* Choose column count so cells are near-square and the grid's overall aspect ratio
     * approximates the display's: cols*rows ~= n and cols/rows ~= displayAspect gives
     * cols ~= sqrt(n * displayAspect). Clamp so we never produce 0 columns or more
     * columns than images. */
    int cols = static_cast<int>(std::round(std::sqrt(n * displayAspect)));
    cols = std::clamp(cols, 1, static_cast<int>(n));
    int rows = static_cast<int>(std::ceil(static_cast<float>(n) / cols));
    float cellHeightPx = static_cast<float>(displayHeightPx) / rows;

    for (int r = 0; r < rows; r++) {
        size_t rowStart = static_cast<size_t>(r) * cols;
        /* The last row may have fewer than `cols` images; dividing the display width by
         * however many images actually land in this row stretches them to fill it,
         * instead of leaving an empty gap where missing columns would have been. */
        int itemsInRow = std::min(static_cast<size_t>(cols), n - rowStart);
        float cellWidthPx = static_cast<float>(displayWidthPx) / itemsInRow;

        for (int c = 0; c < itemsInRow; c++) {
            float x0 = c * cellWidthPx;
            float y0 = r * cellHeightPx;
            float x1 = x0 + cellWidthPx;
            float y1 = y0 + cellHeightPx;

            /* Region's constructor normalizes these pixel coordinates against the
             * display dimensions, matching how ImageMeta::subject_ is normalized. */
            Region region(Point{x0, y0}, Point{x1, y1}, displayWidthPx, displayHeightPx);
            layout.placements().push_back({region, &images[rowStart + c]});
        }
    }

    return layout;
}
