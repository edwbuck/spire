package endpoints

import (
	"container/heap"
	"math/rand"
	"sync"
	"time"
)

type nowFunc func() time.Time

// EventTrackerPQ trackes skipped events using a priority queue. The priority
// queue is ordered by the timestamp of the next poll for each tracked event.
// Each event is polled using the following strategy:
// - for the first minute, polled once per cache poll interval
// - for the next 9 minutes, polled once every 30 seconds
// - for the remainder of the time, polled once every minute.
// An event is polled until it has been observed for longer than the SQL
// transaction timeout period.
type EventTrackerPQ struct {
	// pollInterval is how often the events are polled in the first minute.
	pollInterval time.Duration

	// pollFor is how long the event is polled for.
	pollFor time.Duration

	// jitter, in percent, that is applied to the next poll. If zero, no
	// jitter is applied.
	jitter int

	// nowFn is used for determining the current time.
	// TODO: replace with clock
	nowFn nowFunc

	// queue is the priority queue for events, ordered by the next poll
	// timestamp.
	queue eventQueue

	// pool is used to hold onto the events slice returned from SelectEvents
	// for reuse. Placing the slice in the pool allows the garbage collector
	// to reclaim it between cache poll intervals if needed.
	pool sync.Pool
}

// NewEventTrackerPQ returns a newly initialized event tracker with the given
// pollInterval and sqlTransactionTimeoutInterval.
func NewEventTrackerPQ(pollInterval, pollFor time.Duration, jitter int, nowFn nowFunc) *EventTrackerPQ {
	return &EventTrackerPQ{
		pollInterval: pollInterval,
		pollFor:      pollFor,
		jitter:       jitter,
		nowFn:        nowFn,
		pool: sync.Pool{
			New: func() any {
				return []uint(nil)
			},
		},
	}
}

func (t *EventTrackerPQ) EventCount() uint {
	return uint(t.queue.Len())
}

func (t *EventTrackerPQ) StartTracking(id uint) {
	now := t.nowFn()
	heap.Push(&t.queue, &eventItem{
		id:         id,
		observedAt: now,
		nextPoll:   t.calculateNextPoll(now, now),
	})
}

func (t *EventTrackerPQ) SelectEvents() (events []uint) {
	now := t.nowFn()
	events = t.pool.Get().([]uint)
	// Keep appending events while there are items at the head of the priority
	// queue whose next poll timestamp is at or before now.
	for t.queue.Len() > 0 && t.queue[0].nextPoll.Compare(now) <= 0 {
		head := t.queue[0]

		events = append(events, head.id)

		// Calculate the next polling period. If the returned time is zero then
		// this event has been observed longer than the configured
		// pollFor and no longer needs to be tracked, so pop it
		// from the priority queue.
		//
		// Otherwise, reset the next poll period and fix up the order of the
		// priority queue.
		nextPoll := t.calculateNextPoll(head.observedAt, now)
		if nextPoll.IsZero() {
			heap.Pop(&t.queue)
			continue
		}
		head.nextPoll = nextPoll
		heap.Fix(&t.queue, 0)
	}
	return events
}

func (t *EventTrackerPQ) FreeEvents(events []uint) {
	t.pool.Put(events[:0])
}

// calculateNextPoll calculates the next poll period based on the time
// the event was first observed and now. The next poll period is once per
// poll interval when younger than a minute, every thirty seconds for the next
// 9 minutes, and then once a minute until the event ages out after it has been
// longer than the pollFor since it was first observed.
//
// A 10% jitter is applied to distribute the event polling in the face of
// bursty skipped event observation. E.g. a one minute next poll period will
// result in a value between 57-63 seconds.
func (t *EventTrackerPQ) calculateNextPoll(observedAt time.Time, now time.Time) time.Time {
	var nextPoll time.Duration

	elapsed := now.Sub(observedAt)
	switch {
	case elapsed < time.Minute:
		nextPoll = t.pollInterval
	case elapsed < 10*time.Minute:
		nextPoll = 30 * time.Second
	case elapsed < t.pollFor:
		nextPoll = time.Minute
	default:
		return time.Time{}
	}

	if t.jitter > 0 {
		jitter := nextPoll / time.Duration(t.jitter)
		nextPoll += time.Duration(rand.Int63n(int64(jitter))) - jitter/2
	}
	return now.Add(nextPoll)
}

type eventItem struct {
	id         uint
	observedAt time.Time
	nextPoll   time.Time
}

// eventQueue implements the required interface for using the container/heap
// package for a priority queue. Copied and simplified from the example in the
// heap package.
type eventQueue []*eventItem

func (eq eventQueue) Len() int { return len(eq) }

func (eq eventQueue) Less(i, j int) bool {
	cmp := eq[i].nextPoll.Compare(eq[j].nextPoll)
	switch {
	case cmp < 0:
		return true
	case cmp > 0:
		return false
	default:
		return eq[i].id < eq[j].id
	}
}

func (eq eventQueue) Swap(i, j int) {
	eq[i], eq[j] = eq[j], eq[i]
}

func (eq *eventQueue) Push(x any) {
	*eq = append(*eq, x.(*eventItem))
}

func (eq *eventQueue) Pop() any {
	old := *eq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil // don't stop the GC from reclaiming the item eventually
	*eq = old[:n-1]
	return item
}
