# go-wrr — Smooth Weighted Round-Robin

A generic, deterministic, smooth weighted round-robin scheduler for Go.

Based on the nginx smooth weighted round-robin algorithm. Compiles the
weight distribution into a fixed lookup table at construction time,
making `Next()` an O(1) operation with zero allocation.

## Properties

**Proportional.** Over every `totalWeight` consecutive calls,
each item is returned exactly `Weight()` times.

**Smooth.** Items are interleaved, not batched. An item with
weight W out of total T will never appear more than
`ceil(2W/T)` times consecutively.

**Deterministic.** Two schedulers built from identical inputs
produce identical sequences.

**O(1) dispatch.** Construction is O(N × totalWeight) where N
is the number of items. `Next()` is a single array index plus
modulo.


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


