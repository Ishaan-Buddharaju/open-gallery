#pragma once
/* includes/open_gallery/types.hh */
#include <stdexcept>
#include <vector>

/*
 * Represents a point on an image
 */
struct Point {
    float x;
    float y;
};

/*
 * struct Region(Point topLeft, Point botRight, uint16_t widthPx, uint16_t heightPx)
 *
 * Represents a normalized rectangular region. Used both as the union of all subject
 * boxes within an image (ImageMeta::subject_) and as an image's placement rect within
 * a Layout (see Placement). Normalizes the points based on the dimensions given to the
 * constructor.
 *
 * */
struct Region {
private:
    Point topLeft_;
    Point botRight_;
public:
    /* Constructor Normalizes the points so that it can be used with the aspectRatio */
    Region(Point topLeft, Point botRight, uint16_t widthPx, uint16_t heightPx) {
        if (widthPx <= 0 || heightPx <= 0) {
            throw std::invalid_argument("Invalid image dimensions");
        }

        if (topLeft.x < 0 || topLeft.y < 0 ||
            botRight.x > widthPx ||
            botRight.y > heightPx ||
            topLeft.x > botRight.x ||
            topLeft.y > botRight.y) {
            throw std::invalid_argument("Invalid focal region");
        }
            topLeft_.x = topLeft.x / widthPx;
            topLeft_.y = topLeft.y / heightPx;
            botRight_.x = botRight.x / widthPx;
            botRight.y = botRight.y / heightPx;
        }

    Point topLeft() { return topLeft_; }
    Point botRight() { return botRight_; }

};

/*
 * struct ImageMeta represents a single image.
 * ImageMeta members:
 * - widthPx: original pixel width
 * - heightPx: original pixel height
 * - aspectRatio: float representing the aspect ratio to preserve
 * - subject: pointer to Region object which represents the Union
 * of all subject boxes in the image. Preserved when cropping.
 */

class ImageMeta {
private:
    uint16_t widthPx_;
    uint16_t heightPx_;
    float aspectRatio_;
    Region *subject_;
public:
    /* Constructor derives aspectRatio from the given pixel dimensions */
    ImageMeta(uint16_t widthPx, uint16_t heightPx, Region *subject = nullptr) {
        if (widthPx <= 0 || heightPx <= 0) {
            throw std::invalid_argument("Invalid image dimensions");
        }

        widthPx_ = widthPx;
        heightPx_ = heightPx;
        aspectRatio_ = static_cast<float>(widthPx) / static_cast<float>(heightPx);
        subject_ = subject;
    }

    uint16_t widthPx() const { return widthPx_; }
    uint16_t heightPx() const { return heightPx_; }
    float aspectRatio() const { return aspectRatio_; }
    Region *subject() const { return subject_; }

    void setWidthPx(uint16_t widthPx) {
        if (widthPx <= 0) {
            throw std::invalid_argument("Invalid image dimensions");
        }
        widthPx_ = widthPx;
        aspectRatio_ = static_cast<float>(widthPx_) / static_cast<float>(heightPx_);
    }

    void setHeightPx(uint16_t heightPx) {
        if (heightPx <= 0) {
            throw std::invalid_argument("Invalid image dimensions");
        }
        heightPx_ = heightPx;
        aspectRatio_ = static_cast<float>(widthPx_) / static_cast<float>(heightPx_);
    }

    void setSubject(Region *subject) { subject_ = subject; }
};

/*
 * struct Placement(Region region, const ImageMeta *image)
 *
 * Pairs a non-owning ImageMeta pointer with the Region it occupies within a Layout.
 * Keeps Region itself free of any knowledge of the images it's attached to.
 */
struct Placement {
    Region region;
    const ImageMeta *image;
};

/*
 * struct Layout
 *
 * One candidate gallery layout: the set of images and the Region each is placed in.
 * Stores the entire size of the layout.
 */
class Layout {
private:
    std::vector<Placement> placements_;
    uint16_t widthPx_;
    uint16_t heightPx_;
    float aspectRatio_;
    // TODO Maybe add a generator type struct later
public:
    /* Constructor derives aspectRatio from the given pixel dimensions */
    Layout(uint16_t widthPx, uint16_t heightPx, std::vector<Placement> placements = {}) {
        if (widthPx <= 0 || heightPx <= 0) {
            throw std::invalid_argument("Invalid layout dimensions");
        }

        widthPx_ = widthPx;
        heightPx_ = heightPx;
        aspectRatio_ = static_cast<float>(widthPx) / static_cast<float>(heightPx);
        placements_ = std::move(placements);
    }

    uint16_t widthPx() const { return widthPx_; }
    uint16_t heightPx() const { return heightPx_; }
    float aspectRatio() const { return aspectRatio_; }
    std::vector<Placement>& placements() { return placements_; }
};
