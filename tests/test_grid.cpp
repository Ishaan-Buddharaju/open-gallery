#include <algorithm>
#include <cassert>
#include <cmath>
#include <iostream>
#include <vector>

#include <open_gallery/generators/grid.hh>

namespace {

constexpr float epsilon = 0.0001F;

void assertRect(Region region, float x0, float y0, float x1, float y1) {
    assert(std::fabs(region.topLeft().x - x0) < epsilon);
    assert(std::fabs(region.topLeft().y - y0) < epsilon);
    assert(std::fabs(region.botRight().x - x1) < epsilon);
    assert(std::fabs(region.botRight().y - y1) < epsilon);
}

void testSingleImageFillsDisplay() {
    std::vector<ImageMeta> images{ImageMeta(20, 300)};
    auto layout = generateRowGridLayout(images, 1920, 1080);
    assert(layout.placements().size() == 1);
    assert(layout.placements()[0].image == &images[0]);
    assertRect(layout.placements()[0].region, 0, 0, 1, 1);
}

void testSquareGrid() {
    // Mixed image shapes must still receive equal cells.
    std::vector<ImageMeta> images{
        ImageMeta(100, 100), ImageMeta(400, 100),
        ImageMeta(100, 400), ImageMeta(300, 200)};
    auto layout = generateRowGridLayout(images, 600, 600);
    auto& cells = layout.placements();
    assert(cells.size() == 4);
    assertRect(cells[0].region, 0, 0, 0.5F, 0.5F);
    assertRect(cells[1].region, 0.5F, 0, 1, 0.5F);
    assertRect(cells[2].region, 0, 0.5F, 0.5F, 1);
    assertRect(cells[3].region, 0.5F, 0.5F, 1, 1);
}

void testIncompleteRowFillsWidth() {
    std::vector<ImageMeta> images(5, ImageMeta(100, 100));
    auto layout = generateRowGridLayout(images, 900, 600);
    auto& cells = layout.placements();
    assert(cells.size() == 5);
    assertRect(cells[0].region, 0, 0, 1.0F / 3, 0.5F);
    assertRect(cells[1].region, 1.0F / 3, 0, 2.0F / 3, 0.5F);
    assertRect(cells[2].region, 2.0F / 3, 0, 1, 0.5F);
    assertRect(cells[3].region, 0, 0.5F, 0.5F, 1);
    assertRect(cells[4].region, 0.5F, 0.5F, 1, 1);
}

void testExtremeDisplayShapes() {
    std::vector<ImageMeta> images(4, ImageMeta(100, 100));
    auto wide = generateRowGridLayout(images, 65535, 1);
    auto tall = generateRowGridLayout(images, 1, 65535);
    assert(wide.placements().size() == 4);
    assert(tall.placements().size() == 4);
    for (size_t i = 0; i < images.size(); ++i) {
        float start = static_cast<float>(i) / 4;
        float end = static_cast<float>(i + 1) / 4;
        assertRect(wide.placements()[i].region, start, 0, end, 1);
        assertRect(tall.placements()[i].region, 0, start, 1, end);
    }
}

void testGridInvariants() {
    struct Display { uint16_t width; uint16_t height; };
    for (auto display : {Display{1, 1}, Display{101, 73}, Display{73, 101},
                         Display{1920, 1080}, Display{1080, 1920},
                         Display{65535, 65535}}) {
        for (size_t count = 1; count <= 100; ++count) {
            // Printed on failure by CTest to identify the failing input.
            std::cerr << "Grid: " << count << " images, " << display.width
                      << 'x' << display.height << '\n';
            std::vector<ImageMeta> images(count, ImageMeta(320, 240));
            auto layout = generateRowGridLayout(images, display.width, display.height);
            assert(layout.widthPx() == display.width);
            assert(layout.heightPx() == display.height);
            auto& cells = layout.placements();
            assert(cells.size() == count);
            double area = 0;
            for (size_t i = 0; i < count; ++i) {
                assert(cells[i].image == &images[i]);
                auto a = cells[i].region.topLeft();
                auto b = cells[i].region.botRight();
                assert(std::isfinite(a.x) && std::isfinite(a.y));
                assert(std::isfinite(b.x) && std::isfinite(b.y));
                assert(a.x >= 0 && a.y >= 0 && b.x <= 1 && b.y <= 1);
                assert(a.x < b.x && a.y < b.y);
                area += static_cast<double>(b.x - a.x) * (b.y - a.y);
                for (size_t j = 0; j < i; ++j) {
                    auto c = cells[j].region.topLeft();
                    auto d = cells[j].region.botRight();
                    float overlapX = std::min(b.x, d.x) - std::max(a.x, c.x);
                    float overlapY = std::min(b.y, d.y) - std::max(a.y, c.y);
                    assert(overlapX <= epsilon || overlapY <= epsilon);
                }
            }
            assert(std::fabs(area - 1.0) < epsilon);
        }
    }
}

} // namespace

int main() {
    testSingleImageFillsDisplay();
    testSquareGrid();
    testIncompleteRowFillsWidth();
    testExtremeDisplayShapes();
    testGridInvariants();
}
