package framework

import "testing"

func TestVector2Operations(t *testing.T) {
	got := Vector2AddByVector2AndVector2(NewVector2BySingleAndSingle(2, 3), NewVector2BySingleAndSingle(4, -1))
	want := NewVector2BySingleAndSingle(6, 2)
	if got != want {
		t.Fatalf("got %#v, want %#v", got, want)
	}
}
