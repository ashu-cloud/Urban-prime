package domain

import "testing"

func TestTripStateMachine(t *testing.T) {
	allowed := map[TripStatus][]TripStatus{
		StatusRequested:  {StatusMatching, StatusPaymentFailed, StatusCancelled},
		StatusMatching:   {StatusAssigned, StatusCancelledNoDriver, StatusCancelled},
		StatusAssigned:   {StatusInProgress, StatusCancelled},
		StatusInProgress: {StatusCompleted, StatusCancelled},
	}
	forbidden := []TripStatus{StatusCompleted, StatusCancelled, StatusCancelledNoDriver, StatusPaymentFailed}

	for from, tos := range allowed {
		trip := &Trip{ID: "t1", Status: from}
		for _, to := range tos {
			if err := trip.ValidateTransition(to); err != nil {
				t.Errorf("%s -> %s should be allowed: %v", from, to, err)
			}
		}
	}

	for _, from := range forbidden {
		trip := &Trip{ID: "t1", Status: from}
		if trip.CanTransitionTo(StatusCancelled) {
			t.Errorf("terminal status %s should not transition to CANCELLED", from)
		}
	}

	trip := &Trip{ID: "t1", Status: StatusCompleted}
	if err := trip.ValidateTransition(StatusCancelled); err == nil {
		t.Fatal("completed trip must not cancel")
	}
}

func TestCannotSkipStates(t *testing.T) {
	trip := &Trip{ID: "t1", Status: StatusRequested}
	if trip.CanTransitionTo(StatusCompleted) {
		t.Fatal("must not skip from REQUESTED to COMPLETED")
	}
	if trip.CanTransitionTo(StatusAssigned) {
		t.Fatal("must not skip from REQUESTED to ASSIGNED")
	}
}
