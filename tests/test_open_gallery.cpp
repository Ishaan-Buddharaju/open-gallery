#include <cassert>
#include <cmath>
#include <stdexcept>
#include <vector>

#include <open_gallery/generators/grid.hh>
#include <open_gallery/types.hh>

namespace {

constexpr float kEpsilon = 0.0001F;

void assertNear(float actual, float expected) {
    assert(std::fabs(actual - expected) < kEpsilon);
}

template <typename Func>
void assertThrowsInvalidArgument(Func func) {
    bool threw = false;
    try {
        func();
    } catch (const std::invalid_argument&) {
        threw = true;
    }
    assert(threw);
}

void testRegionNormalizesCoordinates() {
    Region region(Point{25.0F, 10.0F}, Point{75.0F, 40.0F}, 100, 50);

    assertNear(region.topLeft().x, 0.25F);
    assertNear(region.topLeft().y, 0.2F);
    assertNear(region.botRight().x, 0.75F);
    assertNear(region.botRight().y, 0.8F);
}

void testRegionRejectsInvalidCoordinates() {
    assertThrowsInvalidArgument([] {
        Region(Point{-1.0F, 0.0F}, Point{10.0F, 10.0F}, 100, 100);
    });

    assertThrowsInvalidArgument([] {
        Region(Point{0.0F, 0.0F}, Point{101.0F, 10.0F}, 100, 100);
    });

    assertThrowsInvalidArgument([] {
        Region(Point{75.0F, 10.0F}, Point{25.0F, 40.0F}, 100, 100);
    });
}

void testImageMetaMaintainsAspectRatio() {
    Region subject(Point{10.0F, 20.0F}, Point{110.0F, 120.0F}, 200, 200);
    ImageMeta image(400, 200, &subject);

    assert(image.widthPx() == 400);
    assert(image.heightPx() == 200);
    assertNear(image.aspectRatio(), 2.0F);
    assert(image.subject() == &subject);

    image.setHeightPx(400);
    assertNear(image.aspectRatio(), 1.0F);

    image.setWidthPx(800);
    assertNear(image.aspectRatio(), 2.0F);
}

void testGridHandlesEmptyImages() {
    std::vector<ImageMeta> images;
    Layout layout = generateGridLayout(images, 400, 200);

    assert(layout.widthPx() == 400);
    assert(layout.heightPx() == 200);
    assertNear(layout.aspectRatio(), 2.0F);
    assert(layout.placements().empty());
}

void testGridPlacesImagesInNormalizedRegions() {
    std::vector<ImageMeta> images{
        ImageMeta(100, 100),
        ImageMeta(100, 100),
        ImageMeta(100, 100),
        ImageMeta(100, 100),
    };

    Layout layout = generateGridLayout(images, 400, 200);
    auto& placements = layout.placements();

    assert(placements.size() == images.size());
    for (size_t i = 0; i < placements.size(); ++i) {
        assert(placements[i].image == &images[i]);
    }

    assertNear(placements[0].region.topLeft().x, 0.0F);
    assertNear(placements[0].region.topLeft().y, 0.0F);
    assertNear(placements[0].region.botRight().x, 1.0F / 3.0F);
    assertNear(placements[0].region.botRight().y, 0.5F);

    assertNear(placements[3].region.topLeft().x, 0.0F);
    assertNear(placements[3].region.topLeft().y, 0.5F);
    assertNear(placements[3].region.botRight().x, 1.0F);
    assertNear(placements[3].region.botRight().y, 1.0F);
}

void testGridRejectsInvalidDisplayDimensions() {
    std::vector<ImageMeta> images{ImageMeta(100, 100)};

    assertThrowsInvalidArgument([&] {
        generateGridLayout(images, 0, 100);
    });

    assertThrowsInvalidArgument([&] {
        generateGridLayout(images, 100, 0);
    });
}

} // namespace

int main() {
    testRegionNormalizesCoordinates();
    testRegionRejectsInvalidCoordinates();
    testImageMetaMaintainsAspectRatio();
    testGridHandlesEmptyImages();
    testGridPlacesImagesInNormalizedRegions();
    testGridRejectsInvalidDisplayDimensions();

    return 0;
}
