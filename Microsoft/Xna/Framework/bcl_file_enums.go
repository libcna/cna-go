package framework

// This file is CNA-Go language support, not XNA surface.
//
// The three System.IO enums StorageContainer's OpenFile overloads carry. They
// need public Go spellings for the reason System.TimeSpan and
// ReadOnlyCollection<T> do: the pinned contract names them at public signature
// positions, so without a spelling the members that take them cannot be
// projected at all.
//
// # Reference authority
//
//	mscorlib.dll  4.0.30319.1 (RTMRel.030319-0100), assembly version 4.0.0.0
//	sha256 5634668d4775b0113f08ea31093b281fea69bfc4e99227f5ca761b4ed98acc63
//
// # The values are the BCL's, and CNA agrees with them exactly
//
// Every literal below was read from mscorlib and then checked against
// `CNA/C/storage.h`, which defines the same numbers under CNA_FILE_MODE_*,
// CNA_FILE_ACCESS_* and CNA_FILE_SHARE_*. That agreement is why the projection
// passes the value straight through instead of translating it: a translation
// table would be a place for the two to drift.

// FileMode is System.IO.FileMode: how OpenFile treats a file that does or does
// not already exist.
//
// It is NOT a flags enum -- the values are 1 through 6 and combining them is
// meaningless -- so it takes the named-int32 projection every non-flags CLR
// enum gets.
type FileMode int32

const (
	// FileModeCreateNew creates a file and FAILS when one already exists.
	FileModeCreateNew FileMode = 1
	// FileModeCreate creates a file, OVERWRITING an existing one.
	FileModeCreate FileMode = 2
	// FileModeOpen opens an existing file and fails when there is none.
	FileModeOpen FileMode = 3
	// FileModeOpenOrCreate opens the file when it exists and creates it
	// otherwise -- the only mode that cannot fail on existence.
	FileModeOpenOrCreate FileMode = 4
	// FileModeTruncate opens an existing file and truncates it to zero bytes.
	FileModeTruncate FileMode = 5
	// FileModeAppend opens or creates a file and seeks to its end before every
	// write.
	FileModeAppend FileMode = 6
)

// FileAccess is System.IO.FileAccess: what the caller asks to do with the file.
//
// It IS a flags enum in the BCL -- ReadWrite is Read|Write, 1|2 == 3 -- and the
// projection keeps that, because a consumer that ORs the two must get the third.
//
// xna:flags
type FileAccess int32

const (
	// FileAccessRead requests read access.
	FileAccessRead FileAccess = 1
	// FileAccessWrite requests write access.
	FileAccessWrite FileAccess = 2
	// FileAccessReadWrite is Read|Write, and is the value the BCL declares
	// rather than one a consumer has to compose.
	FileAccessReadWrite FileAccess = 3
)

// FileShare is System.IO.FileShare: what a LATER open of the same file is
// allowed to ask for while this one is still open.
//
// A flags enum whose None is a declared zero, so a consumer can name "no
// sharing" rather than writing a bare 0.
//
// xna:flags
type FileShare int32

const (
	// FileShareNone declines sharing: no later open succeeds.
	FileShareNone FileShare = 0
	// FileShareRead allows a later open for reading.
	FileShareRead FileShare = 1
	// FileShareWrite allows a later open for writing.
	FileShareWrite FileShare = 2
	// FileShareReadWrite is Read|Write.
	FileShareReadWrite FileShare = 3
	// FileShareDelete allows the file to be deleted while it is open.
	FileShareDelete FileShare = 4
	// FileShareInheritable makes the handle inheritable by a child process. It
	// is NOT part of the Win32 sharing bits proper -- the BCL adds it -- which
	// is why its value is 0x10 rather than the next bit after Delete.
	FileShareInheritable FileShare = 0x10
)
