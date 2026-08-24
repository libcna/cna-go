package framework

import "testing"

func TestPlayerIndexRawValues(t *testing.T) {
	values := []PlayerIndex{
		PlayerIndexOne,
		PlayerIndexTwo,
		PlayerIndexThree,
		PlayerIndexFour,
	}
	for raw, value := range values {
		if value != PlayerIndex(raw) {
			t.Fatalf("PlayerIndex value %d = %d", raw, value)
		}
	}
}

func TestPlayerIndexPreservesUndefinedInt32Value(t *testing.T) {
	const raw int32 = 12345
	value := PlayerIndex(raw)
	if int32(value) != raw {
		t.Fatalf("undefined PlayerIndex = %d, want %d", value, raw)
	}
}
