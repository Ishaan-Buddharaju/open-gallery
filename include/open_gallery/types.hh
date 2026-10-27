/* includes/open_gallery/types.hh */
#include <stdexcept>

/*
 * Represents a point on an image
 */
struct Point {
    float x;
    float y;
};

/*
 * struct FocalRegion(Point topLeft, Point botRight, uint16_t widthPx, uint16_t heightPx)
 *
 * Used to represent a focal region of an image. This should be used as the union of all regions
 * within an image. Normalizes the point based on the image dimensions given to the constructor.
 *
 * */
struct FocalRegion {
private:
    Point topLeft_;
    Point botRight_;
public:
    /* Constructor Normalizes the points so that it can be used with the aspectRatio */
    FocalRegion(Point topLeft, Point botRight, uint16_t widthPx, uint16_t heightPx) {
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

struct ImageMeta {
    uint64_t id;
    uint16_t widthPx;
    uint16_t heightPx;
    float aspectRatio;
    FocalRegion *subject;
};
