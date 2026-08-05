package device

import (
	"errors"
	"testing"

	"go.autonomous.ai/os/system/domain"
)

func TestReserveAgentRuntimeSwitchRejectsConcurrentSwitch(t *testing.T) {
	svc := &Service{}
	first, err := svc.ReserveAgentRuntimeSwitch(domain.AgentRuntimeSetData{Runtime: "invalid"})
	if err != nil {
		t.Fatalf("reserve first switch: %v", err)
	}

	if _, err := svc.ReserveAgentRuntimeSwitch(domain.AgentRuntimeSetData{Runtime: "openclaw"}); !errors.Is(err, ErrAgentRuntimeSwitchInProgress) {
		t.Fatalf("reserve concurrent switch error = %v, want ErrAgentRuntimeSwitchInProgress", err)
	}

	if _, err := first(); err == nil {
		t.Fatal("first switch with invalid runtime unexpectedly succeeded")
	}

	second, err := svc.ReserveAgentRuntimeSwitch(domain.AgentRuntimeSetData{Runtime: "invalid"})
	if err != nil {
		t.Fatalf("reserve switch after release: %v", err)
	}
	if _, err := second(); err == nil {
		t.Fatal("second switch with invalid runtime unexpectedly succeeded")
	}
}
