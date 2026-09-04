package framework

import "testing"

// TestSystemIOEnumValuesMatchTheBCL pins every literal against the pinned
// mscorlib, one by one.
//
// The adapterConstants list in the verifier admits these NAMES as language
// adapter surface; nothing there checks a value. This test is what does, and it
// is written out longhand so a wrong number has to be typed twice to survive.
func TestSystemIOEnumValuesMatchTheBCL(t *testing.T) {
	for name, got := range map[string]int32{
		"FileMode.CreateNew":    int32(FileModeCreateNew),
		"FileMode.Create":       int32(FileModeCreate),
		"FileMode.Open":         int32(FileModeOpen),
		"FileMode.OpenOrCreate": int32(FileModeOpenOrCreate),
		"FileMode.Truncate":     int32(FileModeTruncate),
		"FileMode.Append":       int32(FileModeAppend),
	} {
		want := map[string]int32{
			"FileMode.CreateNew": 1, "FileMode.Create": 2, "FileMode.Open": 3,
			"FileMode.OpenOrCreate": 4, "FileMode.Truncate": 5, "FileMode.Append": 6,
		}[name]
		if got != want {
			t.Fatalf("%s = %d, want %d", name, got, want)
		}
	}
	if FileAccessRead != 1 || FileAccessWrite != 2 || FileAccessReadWrite != 3 {
		t.Fatalf("FileAccess = %d/%d/%d, want 1/2/3",
			FileAccessRead, FileAccessWrite, FileAccessReadWrite)
	}
	if FileShareNone != 0 || FileShareRead != 1 || FileShareWrite != 2 ||
		FileShareReadWrite != 3 || FileShareDelete != 4 || FileShareInheritable != 16 {
		t.Fatalf("FileShare = %d/%d/%d/%d/%d/%d, want 0/1/2/3/4/16",
			FileShareNone, FileShareRead, FileShareWrite,
			FileShareReadWrite, FileShareDelete, FileShareInheritable)
	}
}

// TestFileAccessAndFileShareCompose pins that the two FLAGS enums really are
// flags: the BCL declares ReadWrite as the OR of its parts rather than as a
// third independent value, and a consumer that composes must land on it.
//
// FileMode is deliberately absent: it is NOT a flags enum -- its values run 1
// through 6 and ORing Open with Append is meaningless -- which is why only two
// of the three carry the xna:flags directive.
func TestFileAccessAndFileShareCompose(t *testing.T) {
	if FileAccessRead|FileAccessWrite != FileAccessReadWrite {
		t.Fatalf("Read|Write = %d, want ReadWrite = %d",
			FileAccessRead|FileAccessWrite, FileAccessReadWrite)
	}
	if FileShareRead|FileShareWrite != FileShareReadWrite {
		t.Fatalf("Read|Write = %d, want ReadWrite = %d",
			FileShareRead|FileShareWrite, FileShareReadWrite)
	}
	// Inheritable is 0x10, not the next bit after Delete, so it does not
	// collide with any combination of the sharing bits proper.
	if FileShareInheritable&(FileShareReadWrite|FileShareDelete) != 0 {
		t.Fatal("Inheritable overlaps the sharing bits; the BCL puts it at 0x10 to avoid exactly that")
	}
	// FileMode's values are consecutive, which is what makes it not a flags
	// enum: Open|Append would be 7, a value the BCL does not declare.
	if FileModeOpen|FileModeAppend == FileModeAppend {
		t.Fatal("FileMode behaved like a flags enum")
	}
}
