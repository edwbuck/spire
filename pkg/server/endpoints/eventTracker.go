package endpoints

import (
	"sync"
	"time"
)

type eventTracker struct {
	pollPeriods uint

	events map[uint]uint

	pool sync.Pool
}

func PollPeriods(pollTime time.Duration, trackTime time.Duration) uint {
	if pollTime < time.Second {
		pollTime = time.Second
	}
	if trackTime < time.Second {
		trackTime = time.Second
	}
	return uint(1 + (trackTime-1)/pollTime)
}

func NewEventTracker(pollPeriods uint) *eventTracker {
	if pollPeriods < 1 {
		pollPeriods = 1
	}

	return &eventTracker{
		pollPeriods: pollPeriods,
		events:      make(map[uint]uint),
		pool: sync.Pool{
			New: func() any {
				return []uint(nil)
			},
		},
	}
}

func (et *eventTracker) PollPeriods() uint {
	return et.pollPeriods
}

func (et *eventTracker) Polls() uint {
	return et.pollPeriods
}

func (et *eventTracker) StartTracking(event uint) {
	et.events[event] = 0
}

func (et *eventTracker) StopTracking(event uint) {
	delete(et.events, event)
}

func (et *eventTracker) SelectEvents() []uint {
	pollList := et.pool.Get().([]uint)
	for event, _ := range et.events {
		if et.events[event] >= et.pollPeriods {
			et.StopTracking(event)
			continue
		}
		pollList = append(pollList, event)
		et.events[event]++
	}
	return pollList
}

func (et *eventTracker) EventCount() uint {
	return uint(len(et.events))
}
