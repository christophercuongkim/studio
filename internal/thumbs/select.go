package thumbs

import (
	"math"
	"sort"
)

// Select picks up to count candidate frames: discard the blurrier half by
// sharpness, then use farthest-point sampling on the timestamp axis to spread
// the picks and avoid near-duplicate moments (plan §14 step 2).
func Select(frames []Frame, count int) []Frame {
	if count <= 0 || len(frames) == 0 {
		return nil
	}
	if len(frames) <= count {
		out := append([]Frame(nil), frames...)
		sort.Slice(out, func(i, j int) bool { return out[i].TimestampSec < out[j].TimestampSec })
		return out
	}

	// Keep the sharpest half (at least `count`).
	bySharp := append([]Frame(nil), frames...)
	sort.Slice(bySharp, func(i, j int) bool { return bySharp[i].Sharpness > bySharp[j].Sharpness })
	keep := max(len(bySharp)/2, count)
	pool := bySharp[:keep]

	// Farthest-point sampling by timestamp: seed with the sharpest frame, then
	// repeatedly add the pool frame whose nearest chosen timestamp is largest.
	chosen := []Frame{pool[0]}
	used := map[int]bool{0: true}
	for len(chosen) < count {
		bestIdx, bestDist := -1, -1.0
		for i, f := range pool {
			if used[i] {
				continue
			}
			d := minTimestampDist(f.TimestampSec, chosen)
			if d > bestDist {
				bestDist, bestIdx = d, i
			}
		}
		if bestIdx < 0 {
			break
		}
		used[bestIdx] = true
		chosen = append(chosen, pool[bestIdx])
	}

	sort.Slice(chosen, func(i, j int) bool { return chosen[i].TimestampSec < chosen[j].TimestampSec })
	return chosen
}

func minTimestampDist(ts float64, chosen []Frame) float64 {
	min := math.Inf(1)
	for _, c := range chosen {
		if d := math.Abs(ts - c.TimestampSec); d < min {
			min = d
		}
	}
	return min
}
