package cna

import "testing"

func TestVector2Operations(t *testing.T) {
	got := NewVector2(2, 3).Add(NewVector2(4, -1)).Scale(2)
	want := NewVector2(12, 4)
	if got != want {
		t.Fatalf("got %#v, want %#v", got, want)
	}
	if NewVector2(3, 4).LengthSquared() != 25 {
		t.Fatal("3-4-5 vector must have squared length 25")
	}
}

func TestKnownColors(t *testing.T) {
	if CornflowerBlue != (Color{R: 100, G: 149, B: 237, A: 255}) {
		t.Fatalf("unexpected CornflowerBlue value: %#v", CornflowerBlue)
	}
}
