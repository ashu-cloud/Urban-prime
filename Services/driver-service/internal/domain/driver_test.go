package domain

import "testing"

func TestDriverStatusTransitions(t *testing.T) {
	d := &Driver{ID: "d1", Status: StatusOffline}
	if err := d.ValidateTransition(StatusAvailable); err != nil {
		t.Fatal(err)
	}
	if d.CanTransitionTo(StatusOnTrip) {
		t.Fatal("offline driver cannot jump to ON_TRIP")
	}

	d.Status = StatusAvailable
	if !d.CanTransitionTo(StatusOnTrip) || !d.CanTransitionTo(StatusOffline) {
		t.Fatal("available driver should go on trip or offline")
	}

	d.Status = StatusOnTrip
	if !d.CanTransitionTo(StatusAvailable) {
		t.Fatal("on-trip driver should return to available")
	}
	if d.CanTransitionTo(StatusOnTrip) {
		t.Fatal("on-trip cannot transition to itself via CanTransitionTo")
	}
}
