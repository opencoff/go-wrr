// wrr_test.go - WRR tests
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
	"testing"
)

// newAsserter returns a func that calls t.Fatalf on failure.
func newAsserter(t *testing.T) func(bool, string, ...any) {
	t.Helper()
	return func(ok bool, msg string, args ...any) {
		t.Helper()
		if !ok {
			t.Fatalf(msg, args...)
		}
	}
}

// --- test type implementing Weighted ---

type wItem struct {
	name string
	w    int
}

func (i wItem) Weight() int { return i.w }

func wi(name string, weight int) wItem {
	return wItem{name: name, w: weight}
}

// run N iterations, return count per name
func tally(w *WRR[wItem], n int) map[string]int {
	m := make(map[string]int)
	for i := 0; i < n; i++ {
		v := w.Next()
		m[v.name]++
	}
	return m
}

func mustNew[T Weighted](z []T) *WRR[T] {
	w, err := New(z)
	if err != nil {
		s := fmt.Sprintf("%s", err)
		panic(s)
	}
	return w
}

// -----------------------------------------------------------
// Basic behavior
// -----------------------------------------------------------

func TestSingleItem(t *testing.T) {
	assert := newAsserter(t)
	w := mustNew([]wItem{wi("A", 5)})

	for i := 0; i < 20; i++ {
		v := w.Next()
		assert(v.name == "A", "expected A, got %s", v.name)
	}
}

func TestEmptyReturnsZero(t *testing.T) {
	assert := newAsserter(t)
	w, err := New([]wItem{})
	assert(err != nil, "expected error, got %v", w)
}

func TestEqualWeights(t *testing.T) {
	assert := newAsserter(t)
	w := mustNew([]wItem{
		wi("A", 1),
		wi("B", 1),
		wi("C", 1),
	})

	m := tally(w, 300)
	assert(m["A"] == 100, "A: expected 100, got %d", m["A"])
	assert(m["B"] == 100, "B: expected 100, got %d", m["B"])
	assert(m["C"] == 100, "C: expected 100, got %d", m["C"])
}

// -----------------------------------------------------------
// Weight proportionality
// -----------------------------------------------------------

func TestWeightRatio3to1(t *testing.T) {
	assert := newAsserter(t)
	w := mustNew([]wItem{
		wi("A", 3),
		wi("B", 1),
	})

	m := tally(w, 400)
	assert(m["A"] == 300, "A: expected 300, got %d", m["A"])
	assert(m["B"] == 100, "B: expected 100, got %d", m["B"])
}

func TestQoSWeights(t *testing.T) {
	assert := newAsserter(t)
	w := mustNew([]wItem{
		wi("P0_I", 3),
		wi("P0_NI", 1),
		wi("P1_I", 2),
		wi("P1_NI", 1),
		wi("P2_I", 1),
		wi("P2_NI", 1),
		wi("P3_I", 1),
		wi("P3_NI", 1),
	})

	rounds := 1100
	m := tally(w, rounds)

	assert(m["P0_I"] == 300, "P0_I: expected 300, got %d", m["P0_I"])
	assert(m["P0_NI"] == 100, "P0_NI: expected 100, got %d", m["P0_NI"])
	assert(m["P1_I"] == 200, "P1_I: expected 200, got %d", m["P1_I"])
	assert(m["P1_NI"] == 100, "P1_NI: expected 100, got %d", m["P1_NI"])
	assert(m["P2_I"] == 100, "P2_I: expected 100, got %d", m["P2_I"])
	assert(m["P2_NI"] == 100, "P2_NI: expected 100, got %d", m["P2_NI"])
	assert(m["P3_I"] == 100, "P3_I: expected 100, got %d", m["P3_I"])
	assert(m["P3_NI"] == 100, "P3_NI: expected 100, got %d", m["P3_NI"])
}

// -----------------------------------------------------------
// Smoothness: no bursts
// -----------------------------------------------------------

func TestSmoothnessNoBurst(t *testing.T) {
	assert := newAsserter(t)
	w := mustNew([]wItem{
		wi("A", 3),
		wi("B", 1),
	})

	maxConsec := 0
	curConsec := 0
	prev := ""

	for i := 0; i < 400; i++ {
		v := w.Next()
		if v.name == prev {
			curConsec++
		} else {
			curConsec = 1
			prev = v.name
		}
		if curConsec > maxConsec {
			maxConsec = curConsec
		}
	}

	assert(maxConsec <= 3,
		"max consecutive picks was %d, expected <= 3",
		maxConsec)
}

func TestSmoothInterleaving8Classes(t *testing.T) {
	assert := newAsserter(t)
	w := mustNew([]wItem{
		wi("P0_I", 3),
		wi("P0_NI", 1),
		wi("P1_I", 2),
		wi("P1_NI", 1),
		wi("P2_I", 1),
		wi("P2_NI", 1),
		wi("P3_I", 1),
		wi("P3_NI", 1),
	})

	totalWeight := 11
	window := totalWeight * 2
	picks := make([]string, 0, 1100)
	for i := 0; i < 1100; i++ {
		picks = append(picks, w.Next().name)
	}

	classes := []string{
		"P0_I", "P0_NI", "P1_I", "P1_NI",
		"P2_I", "P2_NI", "P3_I", "P3_NI",
	}
	for start := 0; start+window <= len(picks); start++ {
		seg := picks[start : start+window]
		for _, cls := range classes {
			found := false
			for _, s := range seg {
				if s == cls {
					found = true
					break
				}
			}
			assert(found,
				"class %s missing in window [%d:%d]",
				cls, start, start+window)
		}
	}
}

// -----------------------------------------------------------
// Stability: deterministic sequence
// -----------------------------------------------------------

func TestDeterministicSequence(t *testing.T) {
	assert := newAsserter(t)
	slots := []wItem{
		wi("A", 5),
		wi("B", 3),
		wi("C", 2),
	}

	w1 := mustNew(slots)
	w2 := mustNew(slots)
	for i := 0; i < 500; i++ {
		a := w1.Next()
		b := w2.Next()
		assert(a.name == b.name,
			"diverged at step %d: %s vs %s", i, a.name, b.name)
	}
}

// -----------------------------------------------------------
// Edge cases
// -----------------------------------------------------------

func TestWeightOfOne(t *testing.T) {
	assert := newAsserter(t)
	w := mustNew([]wItem{
		wi("A", 1),
		wi("B", 1),
		wi("C", 1),
		wi("D", 1),
	})

	for round := 0; round < 50; round++ {
		m := tally(w, 4)
		for _, name := range []string{"A", "B", "C", "D"} {
			assert(m[name] == 1,
				"round %d: %s appeared %d times, expected 1",
				round, name, m[name])
		}
	}
}

func TestLargeWeightDisparity(t *testing.T) {
	assert := newAsserter(t)
	w := mustNew([]wItem{
		wi("heavy", 100),
		wi("light", 1),
	})

	m := tally(w, 10100)
	assert(m["heavy"] == 10000,
		"heavy: expected 10000, got %d", m["heavy"])
	assert(m["light"] == 100,
		"light: expected 100, got %d", m["light"])
}

func TestLargeWeightDisparitySmoothness(t *testing.T) {
	assert := newAsserter(t)
	w := mustNew([]wItem{
		wi("heavy", 100),
		wi("light", 1),
	})

	gap := 0
	maxGap := 0
	for i := 0; i < 10100; i++ {
		v := w.Next()
		if v.name == "light" {
			if gap > maxGap {
				maxGap = gap
			}
			gap = 0
		} else {
			gap++
		}
	}
	assert(maxGap <= 101,
		"light starved for %d picks, expected <= 101",
		maxGap)
}

// -----------------------------------------------------------
// Cycle alignment: exact proportions per full cycle
// -----------------------------------------------------------

func TestExactProportionsPerCycle(t *testing.T) {
	assert := newAsserter(t)
	w := mustNew([]wItem{
		wi("A", 5),
		wi("B", 3),
		wi("C", 2),
	})

	totalWeight := 10
	for cycle := 0; cycle < 100; cycle++ {
		m := tally(w, totalWeight)
		assert(m["A"] == 5,
			"cycle %d: A expected 5, got %d", cycle, m["A"])
		assert(m["B"] == 3,
			"cycle %d: B expected 3, got %d", cycle, m["B"])
		assert(m["C"] == 2,
			"cycle %d: C expected 2, got %d", cycle, m["C"])
	}
}

// -----------------------------------------------------------
// Wraparound: cursor resets cleanly
// -----------------------------------------------------------

func TestWraparound(t *testing.T) {
	assert := newAsserter(t)
	w := mustNew([]wItem{
		wi("A", 2),
		wi("B", 1),
	})

	first := make([]string, 3)
	for i := range first {
		first[i] = w.Next().name
	}

	// Next 3 should be identical (cursor wrapped)
	for i := 0; i < 3; i++ {
		v := w.Next()
		assert(v.name == first[i],
			"wraparound mismatch at %d: expected %s, got %s",
			i, first[i], v.name)
	}
}
