package contextview

import "testing"

func TestContractLimitsRemainTokenBounded(t *testing.T) {
	if DefaultPage > MaxPage || DefaultRead > MaxReadBytes || MaxReadLines != 400 || DefaultTreeEntries > MaxTreeEntries || DefaultTreeDepth > MaxTreeDepth {
		t.Fatalf("invalid context bounds: page=%d/%d read=%d/%d lines=%d", DefaultPage, MaxPage, DefaultRead, MaxReadBytes, MaxReadLines)
	}
}
