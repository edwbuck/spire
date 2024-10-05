package endpoints

import (
	"testing"
	"time"

	"github.com/bmizerany/perks/quantile"
)

type eventTrackerUnderTest interface {
	StartTracking(uint)
	SelectEvents() []uint
	FreeEvents([]uint)
	EventCount() uint
}

const (
	// How many seconds in the day
	secondsPerDay = 60 * 60 * 24

	// How many events we want to simulate arriving over a one day period.
	eventsPerDay = 800_000

	// How many reload intervals in a day (assuming one every 5s)
	reloadsPerDay = secondsPerDay / 5

	// How many events would arrive every reload interval assuming an
	// even distribution of events.
	eventsPerReloadInterval = eventsPerDay / reloadsPerDay
)

func BenchmarkEventTrackerCurrent(b *testing.B) {
	var (
		pollPeriods    = PollPeriods(defaultCacheReloadInterval, defaultSQLTransactionTimeout)
		pollBoundaries = BoundaryBuilder(defaultCacheReloadInterval, defaultSQLTransactionTimeout)
	)
	benchmarkEventTracker(b, func(nowFn nowFunc) eventTrackerUnderTest {
		return NewEventTracker(pollPeriods, pollBoundaries)
	})
}

func BenchmarkEventTrackerPriorityQueue(b *testing.B) {
	benchmarkEventTracker(b, func(nowFn nowFunc) eventTrackerUnderTest {
		return NewEventTrackerPQ(defaultCacheReloadInterval, defaultSQLTransactionTimeout, 10, nowFn)
	})
}

func benchmarkEventTracker(b *testing.B, newEventTracker func(nowFunc) eventTrackerUnderTest) {
	clk := newFakeClock()
	eventTracker := newEventTracker(clk.Now)
	q := quantile.NewTargeted(0.50, 0.95, 0.99)

	var nextEvent uint

	// Each iteration simulates a reload interval where:
	// 1. some number of new skipped events are tracked
	// 2. the events selected for polling are determined
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Track the newly observed skipped events
		for j := 0; j < eventsPerReloadInterval; j++ {
			nextEvent += 2
			eventTracker.StartTracking(nextEvent)
		}

		clk.Advance(defaultCacheReloadInterval)

		// Select events to be polled
		selected := eventTracker.SelectEvents()

		// Record how many were selected
		q.Insert(float64(len(selected)))

		// Give them back to the tracker
		eventTracker.FreeEvents(selected)
	}
	b.StopTimer()

	// Afterwards display measurements of how many events were tracked
	// and percentiles of how many events were selected each reload
	// interval.
	q50 := q.Query(0.50)
	q95 := q.Query(0.95)
	q99 := q.Query(0.99)
	b.Logf("    eventCount              : %d", eventTracker.EventCount())
	b.Logf("    pollsPerInterval(50th)  : %.02f (%.02f)", q50, q50/float64(eventTracker.EventCount()))
	b.Logf("    pollsPerInterval(95th)  : %.02f (%.02f)", q95, q95/float64(eventTracker.EventCount()))
	b.Logf("    pollsPerInterval(99th)  : %.02f (%.02f)", q99, q99/float64(eventTracker.EventCount()))
}

type fakeClock struct {
	start time.Time
	now   time.Time
}

func newFakeClock() *fakeClock {
	start := time.Now().Truncate(time.Hour)
	return &fakeClock{start: start, now: start}
}

func (c *fakeClock) Now() time.Time {
	return c.now
}

func (c *fakeClock) Advance(d time.Duration) {
	c.now = c.now.Add(d)
}

func (c *fakeClock) Elapsed() time.Duration {
	return c.now.Sub(c.start)
}
