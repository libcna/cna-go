package graphics

import "testing"

func TestClearOptionsRawValuesAndUnnamedZero(t *testing.T) {
	if ClearOptionsTarget != 1 || ClearOptionsDepthBuffer != 2 || ClearOptionsStencil != 4 {
		t.Fatal("ClearOptions constants do not match XNA")
	}
	var zero ClearOptions
	if int32(zero) != 0 {
		t.Fatalf("zero ClearOptions = %d, want raw 0", zero)
	}
}

func TestClearOptionsDeclaredCombinations(t *testing.T) {
	tests := []struct {
		name string
		got  ClearOptions
		want ClearOptions
	}{
		{"target-depth", ClearOptionsTarget | ClearOptionsDepthBuffer, ClearOptions(3)},
		{"target-stencil", ClearOptionsTarget | ClearOptionsStencil, ClearOptions(5)},
		{"depth-stencil", ClearOptionsDepthBuffer | ClearOptionsStencil, ClearOptions(6)},
		{"all-declared", ClearOptionsTarget | ClearOptionsDepthBuffer | ClearOptionsStencil, ClearOptions(7)},
	}
	for _, test := range tests {
		if test.got != test.want {
			t.Errorf("%s combination = %d, want %d", test.name, test.got, test.want)
		}
	}
}

func TestClearOptionsRawDomainAndBitwiseInteraction(t *testing.T) {
	for _, raw := range []int32{0, 8, 1 << 20, -1} {
		if got := int32(ClearOptions(raw)); got != raw {
			t.Fatalf("ClearOptions(%d) = %d", raw, got)
		}
	}
	if got := ClearOptions(8) | ClearOptionsTarget; got != ClearOptions(9) {
		t.Fatalf("ClearOptions(8) | Target = %d, want 9", got)
	}
	if got := ClearOptions(7) & ClearOptionsStencil; got != ClearOptionsStencil {
		t.Fatalf("ClearOptions(7) & Stencil = %d, want 4", got)
	}
	if got := ClearOptions(2) & ClearOptionsTarget; got != ClearOptions(0) {
		t.Fatalf("ClearOptions(2) & Target = %d, want raw 0", got)
	}
}
