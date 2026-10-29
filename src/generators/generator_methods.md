# Generator Methods

## Spaced Grid: 
I preserve constant spacing while centering smaller images. 
### Determining Candidate Cell Fit
For layout with W_i x H_i
and 
For candidate cells W_c x H_c

scale_min > max(W_c / W_i, H_c / H_i)
scale_max > max(W_c / W_f, H_c / H_f ) where f denotes the focal region

Images fit into a cell candidate iff scale_min <=scale_max

### Generating Grid Candidates
For Spaced Grid we constrain it to a rectangualar grid.
I considered restricting grids to compositions of layouts that aren't rows (1 x K), but prime numbers of images violate this property. The workaround of allowing non-rectangulars would overstep on a row distribution method. I'd rather just hand off the prime case to both the row and row distribution generator and invalidate grid. 

Therefore, this layout maintains the invariant that the number of images is non prime.
