# go-wrr — Smooth Weighted Round-Robin

This package implements a smooth, weighted round-robin scheduler.

It provides a high-performance, concurrency-safe scheduler that distributes
items according to their integer weights. Unlike naive weighted round-robin
(which can generate "bursts" of the same item), this package implements the
"smooth" algorithm used by Nginx. This ensures that items are interleaved
evenly over time while strictly adhering to the configured weight ratios.

## Key Features

- Smooth Distribution: Spreads high-weight items evenly across the sequence
  (e.g., "A A B" becomes "A B A").
- O(1) Runtime: The selection logic involves a single atomic increment and
  array lookup, making it suitable for high-throughput hot paths.
- Deterministic: The sequence is precompiled and cycles deterministically.
- Concurrency Safe: Safe for concurrent access by multiple goroutines without
  mutex locking during selection.

## Algorithm Details

The scheduler precompiles a lookup table based on the greatest common divisor
(GCD) of the input weights. This optimization significantly reduces memory usage
for proportionate weights (e.g., weights {100, 200} result in a table of size 3,
not 300).


## Install

```
go get github.com/opencoff/go-wrr
```

## Usage

* Implement your queuing object to conform to `wrr.Weighted` interface
* Define weights for each object
* Instantiate WRR instance by calling `wrr.New()`
* Call `Next()` to dispatch

### Example

```go
type Queue struct {
    Name string
    Ch   chan *Request
    w    int
}

// implement the wrr.Weighted interface
func (q Queue) Weight() int { return q.w }

queues := []Queue{
    {Name: "critical", w: 3},
    {Name: "normal",   w: 1},
}

sched, err := wrr.New(queues)

for {
    q := sched.Next()
    // dispatch from q.Ch
}
```

