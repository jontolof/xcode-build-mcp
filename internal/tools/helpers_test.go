package tools

import (
	"testing"
)

func TestSelectBestSimulator(t *testing.T) {
	// This test checks what selectBestSimulator actually returns
	// Behavior depends on whether simulators are available
	simulator, err := selectBestSimulator("")

	t.Logf("selectBestSimulator returned: simulator=%+v, err=%v", simulator, err)

	// Either succeeds with a simulator or fails with an error
	if err == nil {
		// Success case - simulator should be valid
		if simulator == nil {
			t.Error("Expected non-nil simulator when no error occurs")
		}
		if simulator.UDID == "" {
			t.Error("Expected non-empty UDID in successful result")
		}
		t.Logf("Success: Found simulator %s (%s)", simulator.Name, simulator.UDID)
	} else {
		// Failure case - simulator should be nil
		if simulator != nil {
			t.Errorf("Expected nil simulator when error occurs, got: %+v", simulator)
		}
		t.Logf("Expected failure in environment without simulators: %v", err)
	}
}
