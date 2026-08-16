package framework

import "testing"

func TestVector2Operations(t *testing.T) {
	got := NewVector2(2, 3).Add(NewVector2(4, -1))
	want := NewVector2(6, 2)
	if got != want {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}
