#pragma once
#include <vector>
#include <open_gallery/types.hh>

Layout generateRowGridLayout(const std::vector<ImageMeta>& images, uint16_t displayWidthPx, uint16_t displayHeightPx);
