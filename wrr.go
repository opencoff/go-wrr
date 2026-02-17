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

// Package wrr implements a smooth, weighted round-robin scheduler.
//
// It provides a high-performance, concurrency-safe scheduler that distributes
// items according to their integer weights. Unlike naive weighted round-robin
// (which can generate "bursts" of the same item), this package implements the
// "smooth" algorithm used by Nginx. This ensures that items are interleaved
// evenly over time while strictly adhering to the configured weight ratios.
//
// Key Features:
//
//   - Smooth Distribution: Spreads high-weight items evenly across the sequence
//     (e.g., "A A B" becomes "A B A").
//   - O(1) Runtime: The selection logic involves a single atomic increment and
//     array lookup, making it suitable for high-throughput hot paths.
//   - Deterministic: The sequence is precompiled and cycles deterministically.
//   - Concurrency Safe: Safe for concurrent access by multiple goroutines without
//     mutex locking during selection.
//
// Algorithmic Details:
//
// The scheduler precompiles a lookup table based on the greatest common divisor
// (GCD) of the input weights. This optimization significantly reduces memory usage
// for proportionate weights (e.g., weights {100, 200} result in a table of size 3,
// not 300).
//
// Usage:
//
//	type Server struct {
//		Name string
//		W    int
//	}
//
//	func (s Server) Weight() int { return s.W }
//
//	func main() {
//		servers := []Server{
//			{Name: "s1", W: 5},
//			{Name: "s2", W: 1},
//		}
//
//		sched, err := wrr.New(servers)
//		if err != nil {
//			log.Fatal(err)
//		}
//
//		// fast, thread-safe selection
//		s := sched.Next()
//	}
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
	slots []T
	seq   []uint16
	next  atomic.Uint64
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
	if n >= 65536 {
		return nil, fmt.Errorf("wrr: too many WRR slots (%d)", n)
	}

	tot := 0

	// single big alloc to reduce gc pressure
	blk := make([]int, 2*n)

	// eff: effective weights (scaled by gcd)
	eff, cur := blk[:n], blk[n:]
	for i := range slots {
		s := slots[i]
		w := s.Weight()
		if w <= 0 {
			return nil, fmt.Errorf("wrr: slot index %d: bad weight %d", i, w)
		}
		eff[i] = w
		tot += w
	}

	// Calculate the gcd and scale the weights so we don't have explosion of slots
	eff, tot = normalize(eff, tot)

	// hold short indices instead of 'T'
	seq := make([]uint16, tot)

	// now populate the fast lookup table
	for i := range seq {
		var best int
		for j := range eff {
			cur[j] += eff[j]
			if cur[j] > cur[best] {
				best = j
			}
		}
		seq[i] = uint16(best)
		cur[best] -= tot
	}

	w := &WRR[T]{
		slots: make([]T, n),
		seq:   seq,
	}

	copy(w.slots, slots)
	return w, nil
}

// Returns the next item in the smooth weighted sequence.
// Cycles deterministically in O(1) and is concurrency-safe.
func (w *WRR[T]) Next() T {
	i := (w.next.Add(1) - 1) % uint64(len(w.seq))
	j := w.seq[i]
	return w.slots[j]
}

func gcd(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}

// normalize the weights by reducing with the gcd of all the weights.
// this reduces the total size of the seq slice
func normalize(w []int, tot int) ([]int, int) {
	g := w[0]

	for _, z := range w[1:] {
		g = gcd(g, z)
	}

	if g > 1 {
		tot = 0
		for i := range w {
			w[i] /= g
			tot += w[i]
		}
	}

	return w, tot
}
