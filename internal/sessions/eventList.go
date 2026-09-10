package sessions

import (
	"iter"

	"google.golang.org/adk/v2/session"
)

// eventList adapts []*session.Event to session.Events.
type eventList struct {
	items []*session.Event
}

func (el *eventList) All() iter.Seq[*session.Event] {
	return func(yield func(*session.Event) bool) {
		for _, ev := range el.items {
			if !yield(ev) {
				return
			}
		}
	}
}

func (el *eventList) Len() int {
	return len(el.items)
}

func (el *eventList) At(i int) *session.Event {
	if i >= 0 && i < len(el.items) {
		return el.items[i]
	}
	return nil
}
