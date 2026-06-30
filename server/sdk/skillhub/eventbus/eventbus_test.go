package eventbus_test

import (
	"context"
	"testing"

	"miqro-skillhub/server/sdk/skillhub/eventbus"
)

type testEvent struct {
	name string
}

func (e testEvent) EventName() string { return e.name }

func TestNoopBus_NotRecorded(t *testing.T) {
	bus := eventbus.NewNoopBus(false)
	_ = bus.Publish(context.Background(), testEvent{name: "test"})

	// The slice must remain nil because recorded=false.
	if bus.Events != nil {
		t.Errorf("expected nil Events, got %v", bus.Events)
	}
}

func TestNoopBus_Recorded(t *testing.T) {
	bus := eventbus.NewNoopBus(true)
	_ = bus.Publish(context.Background(), testEvent{name: "ev1"})
	_ = bus.Publish(context.Background(), testEvent{name: "ev2"})

	if len(bus.Events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(bus.Events))
	}
	if bus.Events[0].EventName() != "ev1" {
		t.Errorf("expected ev1, got %s", bus.Events[0].EventName())
	}
	if bus.Events[1].EventName() != "ev2" {
		t.Errorf("expected ev2, got %s", bus.Events[1].EventName())
	}
}

func TestNoopBus_Reset(t *testing.T) {
	bus := eventbus.NewNoopBus(true)
	_ = bus.Publish(context.Background(), testEvent{name: "ev"})
	bus.Reset()

	if len(bus.Events) != 0 {
		t.Errorf("expected 0 events after reset, got %d", len(bus.Events))
	}
}

func TestNoopBus_NeverReturnsError(t *testing.T) {
	bus := eventbus.NewNoopBus(true)
	if err := bus.Publish(context.Background(), testEvent{name: "x"}); err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}
