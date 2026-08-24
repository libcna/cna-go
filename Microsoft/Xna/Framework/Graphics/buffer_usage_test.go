package graphics

import "testing"

func TestBufferUsageRawValuesAndZeroValue(t *testing.T) {
	if BufferUsageNone != 0 || BufferUsageWriteOnly != 1 {
		t.Fatal("BufferUsage constants do not match XNA")
	}

	var zero BufferUsage
	if zero != BufferUsageNone {
		t.Fatalf("zero BufferUsage = %d, want None", zero)
	}
}

func TestBufferUsageRawDomainAndBitComposition(t *testing.T) {
	if got := BufferUsageNone | BufferUsageWriteOnly; got != BufferUsageWriteOnly {
		t.Fatalf("None | WriteOnly = %d, want WriteOnly", got)
	}
	if got := BufferUsageWriteOnly | BufferUsageWriteOnly; got != BufferUsageWriteOnly {
		t.Fatalf("WriteOnly | WriteOnly = %d, want WriteOnly", got)
	}
	if got := BufferUsage(2) | BufferUsageWriteOnly; got != BufferUsage(3) {
		t.Fatalf("BufferUsage(2) | WriteOnly = %d, want 3", got)
	}

	for _, raw := range []int32{2, 3, 1 << 20, -1} {
		if got := int32(BufferUsage(raw)); got != raw {
			t.Fatalf("BufferUsage(%d) = %d", raw, got)
		}
	}
}
