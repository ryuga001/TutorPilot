package lecture

import "testing"

func TestCanTransitionFullMatrix(t *testing.T) {
	states := []string{StatusScheduled, StatusLive, StatusEnded, StatusCancelled}
	allowed := map[string]map[string]bool{
		StatusScheduled: {StatusLive: true, StatusCancelled: true},
		StatusLive:      {StatusEnded: true},
		StatusEnded:     {},
		StatusCancelled: {},
	}
	for _, from := range states {
		for _, to := range states {
			want := allowed[from][to]
			if got := CanTransition(from, to); got != want {
				t.Errorf("CanTransition(%q, %q) = %v, want %v", from, to, got, want)
			}
		}
	}
}

func TestTerminalStatesAllowNothing(t *testing.T) {
	for _, from := range []string{StatusEnded, StatusCancelled} {
		for _, to := range []string{StatusScheduled, StatusLive, StatusEnded, StatusCancelled} {
			if CanTransition(from, to) {
				t.Errorf("CanTransition(%q, %q) = true, want terminal", from, to)
			}
		}
	}
}

func TestLiveLectureCannotBeCancelled(t *testing.T) {
	if CanTransition(StatusLive, StatusCancelled) {
		t.Error("a live lecture must be ended, not cancelled")
	}
}

func TestEndedLectureCannotRestart(t *testing.T) {
	if CanTransition(StatusEnded, StatusLive) {
		t.Error("restarting an ended lecture would launch a second recording against a closed room")
	}
}

func TestNoSelfTransitions(t *testing.T) {
	for _, s := range []string{StatusScheduled, StatusLive, StatusEnded, StatusCancelled} {
		if CanTransition(s, s) {
			t.Errorf("CanTransition(%q, %q) = true; the conditional UPDATE needs a real state change", s, s)
		}
	}
}

func TestUnknownStatesRefused(t *testing.T) {
	if CanTransition("bogus", StatusLive) {
		t.Error("unknown source state accepted")
	}
	if CanTransition(StatusScheduled, "bogus") {
		t.Error("unknown target state accepted")
	}
}
