// wrr.go - weighted round robin implementation
//
// (c) 2024 Sudhi Herle <sw-at-herle.net>
//
// Copyright 2024- Sudhi Herle <sw-at-herle-dot-net>
// License: BSD-2-Clause
//
// If you need a commercial license for this work, please contact
// the author.
//
// This software does not come with any express or implied
// warranty; it is provided "as is". No claim  is made to its
// suitability for any purpose.

package wrr

import (
	"fmt"
	"sync/atomic"
)

// Weighted is the constraint for schedulable items.
type Weighted interface {
	Weight() int
}

// WRR is a precompiled smooth weighted round-robin scheduler.
// Safe for concurrent use.
type WRR[T Weighted] struct {
	seq  []T
	next atomic.Uint64
}

// Constructs a new scheduler from the given slots. Each slot's
// `Weight()` determines its share of selections. The weight
// distribution is compiled into a lookup table at construction
// time.
//
// The input slice is not retained or modified.
//
// Returns a scheduler where `Next()` is O(1) and returns nil
// on error
func New[T Weighted](slots []T) (*WRR[T], error) {
	n := len(slots)

	if n == 0 {
		return nil, fmt.Errorf("wrr: no slots to weight")
	}

	tot := 0
	for i := range slots {
		s := slots[i]
		w := s.Weight()
		if w < 0 {
			return nil, fmt.Errorf("wrr: slot index %d: weight %d", i, w)
		}
		tot += w
	}

	seq := make([]T, tot)
	cur := make([]int, n)

	// now populate the fast lookup table
	for i := range seq {
		var best int
		for j := range slots {
			s := slots[j]
			cur[j] += s.Weight()
			if cur[j] > cur[best] {
				best = j
			}
		}
		seq[i] = slots[best]
		cur[best] -= tot
	}

	w := &WRR[T]{
		seq: seq,
	}
	return w, nil
}

// Returns the next item in the smooth weighted sequence.
// Cycles deterministically in O(1) and is concurrency-safe.
func (w *WRR[T]) Next() T {
	j := (w.next.Add(1) - 1) % uint64(len(w.seq))
	return w.seq[j]
}
