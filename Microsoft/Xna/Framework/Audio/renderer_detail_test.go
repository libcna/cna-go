package audio

import (
	"testing"

	"github.com/openeggbert/cna-go/internal/bclhash"
)

// rendererDetailStringHash names the pinned mscorlib algorithm at the test's
// own call sites, so a test that started computing a DIFFERENT hash than the
// projection would be visible here rather than agreeing with itself.
func rendererDetailStringHash(value string) int32 { return bclhash.String(value) }

// RendererDetail is two strings and five members, and every one of them has a
// detail that a plausible projection would get wrong. These tests pin each.

// TestRendererDetailReadsItsTwoFields pins the accessors and the zero value,
// which is the only value a consumer can build today: the constructor is
// `assembly` in the reference and unexported here.
func TestRendererDetailReadsItsTwoFields(t *testing.T) {
	var zero RendererDetail
	if zero.FriendlyName() != "" || zero.RendererId() != "" {
		t.Fatalf("default(RendererDetail) = %q/%q, want two empty strings",
			zero.FriendlyName(), zero.RendererId())
	}
	detail := newRendererDetail("Speakers", "{0.0.0.00000000}")
	if detail.FriendlyName() != "Speakers" || detail.RendererId() != "{0.0.0.00000000}" {
		t.Fatalf("RendererDetail = %q/%q", detail.FriendlyName(), detail.RendererId())
	}
}

// TestRendererDetailEqualityComparesBothStrings pins op_Equality's two-field
// test and its short circuit, and pins that op_Inequality is its negation
// rather than an independent body.
func TestRendererDetailEqualityComparesBothStrings(t *testing.T) {
	for _, row := range []struct {
		name        string
		left, right RendererDetail
		want        bool
	}{
		{"identical", newRendererDetail("a", "b"), newRendererDetail("a", "b"), true},
		{"both zero", RendererDetail{}, RendererDetail{}, true},
		{"name differs", newRendererDetail("a", "b"), newRendererDetail("z", "b"), false},
		{"id differs", newRendererDetail("a", "b"), newRendererDetail("a", "z"), false},
		{"both differ", newRendererDetail("a", "b"), newRendererDetail("z", "y"), false},
		// The two fields are compared SEPARATELY, so a pair whose fields are
		// swapped is unequal even though the multiset of strings matches. A
		// projection that compared a concatenation would pass every row above
		// and fail this one.
		{"fields swapped", newRendererDetail("a", "b"), newRendererDetail("b", "a"), false},
		// The two fields are compared SEPARATELY and not as a joined string.
		// These two concatenate to the same "abc", so a projection that
		// compared left.name+left.id would call them equal -- and every other
		// row here would still pass. This is the row that distinguishes them.
		{"same concatenation, different split", newRendererDetail("ab", "c"), newRendererDetail("a", "bc"), false},
	} {
		if got := RendererDetailOperatorEqualityByRendererDetailAndRendererDetail(row.left, row.right); got != row.want {
			t.Fatalf("%s: op_Equality = %v, want %v", row.name, got, row.want)
		}
		if got := RendererDetailOperatorInequalityByRendererDetailAndRendererDetail(row.left, row.right); got != !row.want {
			t.Fatalf("%s: op_Inequality = %v, want %v", row.name, got, !row.want)
		}
	}
}

// TestRendererDetailEqualsIsTypeExact pins Equals(object)'s middle guard: the
// reference compares the two RUNTIME types with Type::op_Inequality, so
// anything that is not a RendererDetail answers false instead of throwing at
// the unbox.
func TestRendererDetailEqualsIsTypeExact(t *testing.T) {
	detail := newRendererDetail("a", "b")
	if !detail.Equals(newRendererDetail("a", "b")) {
		t.Fatal("Equals refused an equal RendererDetail")
	}
	if detail.Equals(newRendererDetail("a", "z")) {
		t.Fatal("Equals accepted a RendererDetail with a different id")
	}
	// The null branch, which the reference tests FIRST.
	if detail.Equals(nil) {
		t.Fatal("Equals(null) answered true")
	}
	// Every other type answers false rather than panicking.
	for _, other := range []any{"a", 0, struct{ name, id string }{"a", "b"}, &detail} {
		if detail.Equals(other) {
			t.Fatalf("Equals accepted a %T", other)
		}
	}
	// The ZERO value is what makes the type guard load-bearing. A projection
	// that ignored the assertion's ok and compared against the zero
	// RendererDetail a failed assertion yields would answer TRUE here, and
	// would answer correctly for every non-zero receiver above -- so this is
	// the receiver that distinguishes a real type test from a silent default.
	var zero RendererDetail
	for _, other := range []any{nil, "a", 0, RendererDetail{}.ToString()} {
		if zero.Equals(other) {
			t.Fatalf("a zero RendererDetail claimed equality with a %T", other)
		}
	}
	if !zero.Equals(RendererDetail{}) {
		t.Fatal("a zero RendererDetail is not equal to another zero one")
	}
}

// TestRendererDetailHashXorsTheTwoStringHashes pins GetHashCode's body,
// including the detail that decides it: the guard is IsNullOrEmpty, so an EMPTY
// string contributes ZERO rather than String.GetHashCode("").
func TestRendererDetailHashXorsTheTwoStringHashes(t *testing.T) {
	// The zero value hashes to zero, because both halves are empty.
	if got := (RendererDetail{}).GetHashCode(); got != 0 {
		t.Fatalf("default(RendererDetail).GetHashCode() = %d, want 0", got)
	}
	// A detail with only a name hashes to that name's hash alone, which is what
	// XOR with zero means -- and proves the empty half contributed nothing.
	nameOnly := newRendererDetail("Speakers", "")
	if got, want := nameOnly.GetHashCode(), rendererDetailStringHash("Speakers"); got != want {
		t.Fatalf("name-only hash = %d, want the name's own hash %d", got, want)
	}
	idOnly := newRendererDetail("", "Speakers")
	if got := idOnly.GetHashCode(); got != nameOnly.GetHashCode() {
		t.Fatal("the two halves are not symmetric under XOR")
	}
	// And the full value is the XOR of the two.
	both := newRendererDetail("Speakers", "Headphones")
	want := rendererDetailStringHash("Speakers") ^ rendererDetailStringHash("Headphones")
	if got := both.GetHashCode(); got != want {
		t.Fatalf("GetHashCode = %d, want %d", got, want)
	}
	// Equal values hash equally, which is the property the CLR contract
	// actually requires of the pair.
	if newRendererDetail("a", "b").GetHashCode() != newRendererDetail("a", "b").GetHashCode() {
		t.Fatal("equal RendererDetails hash differently")
	}
	// A value whose two strings are the SAME hashes to zero, because x^x is 0.
	// That is a real property of the reference's body and not a defect: it is
	// what makes the empty-string case above indistinguishable from a collision.
	if got := newRendererDetail("same", "same").GetHashCode(); got != 0 {
		t.Fatalf("a detail whose halves match hashes to %d; x^x is 0", got)
	}
}

// TestRendererDetailToStringIsTheTypeName pins the member that reports NOTHING
// about the value. The reference boxes and calls ValueType::ToString, which
// answers GetType().ToString() -- so a reader expecting the friendly name here
// is reading the wrong member.
func TestRendererDetailToStringIsTheTypeName(t *testing.T) {
	const want = "Microsoft.Xna.Framework.Audio.RendererDetail"
	if got := (RendererDetail{}).ToString(); got != want {
		t.Fatalf("default(RendererDetail).ToString() = %q, want %q", got, want)
	}
	if got := newRendererDetail("Speakers", "id").ToString(); got != want {
		t.Fatalf("ToString() = %q; it does NOT report the fields", got)
	}
}
