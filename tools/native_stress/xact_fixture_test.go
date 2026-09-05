package main

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// The three fixture writers are the only reason the XACT slice can measure
// anything, and a wrong header does not fail loudly: XACT's parsers report
// through CNA's LOG, so a malformed file becomes a refusal the slice records
// and moves past. These tests pin the layout so a silent regression is a
// failing test instead of a quietly skipped slice.

func TestTheSettingsFixtureHasTheHeaderTheParserSeeksPast(t *testing.T) {
	data := xactSettings()
	if got := string(data[:4]); got != "XGSF" {
		t.Fatalf("the settings magic is %q, want XGSF", got)
	}
	// 80 is the minimum the parser will look past -- measured before any of
	// this was bound, by growing a header two bytes at a time until it stopped
	// answering "XGS: file too small". Every table offset below is absolute, so
	// the first one starting at 80 is what makes the header's padding load
	// bearing rather than cosmetic.
	if len(data) != 136 {
		t.Fatalf("the settings fixture is %d bytes, want 136", len(data))
	}
	if got := binary.LittleEndian.Uint16(data[4:6]); got != 46 {
		t.Fatalf("the settings contentVersion is %d, want 46", got)
	}
	if got := binary.LittleEndian.Uint16(data[19:21]); got != 2 {
		t.Fatalf("the settings categoryCount is %d, want 2", got)
	}
	if got := binary.LittleEndian.Uint32(data[33:37]); got != 80 {
		t.Fatalf("the category table starts at %d, want 80", got)
	}
	// The variable's accessibility byte is what decides whether the slice's
	// SetGlobalVariable round trip proves the route or proves nothing. 0x01 is
	// PUBLIC; 0x03 would be PUBLIC|READONLY, and a read-only variable accepts
	// every write and ignores it.
	if data[100] != 0x01 {
		t.Fatalf("the global variable's accessibility byte is %#02x, want 0x01 (PUBLIC, not read-only)", data[100])
	}
	for _, name := range []string{"Music\x00", "SFX\x00", "SpeedOfSound\x00"} {
		if !bytes.Contains(data, []byte(name)) {
			t.Fatalf("the settings fixture has no %q", name)
		}
	}
}

func TestTheWaveBankFixtureAddressesItsWaveDataWhereItsHeaderSaysItIs(t *testing.T) {
	first := xactSine(440, 0.01)
	second := xactSine(660, 0.01)
	data := xactWaveBank("Waves", [][]byte{first, second})
	if got := string(data[:4]); got != "WBND" {
		t.Fatalf("the wave bank magic is %q, want WBND", got)
	}
	if got := binary.LittleEndian.Uint32(data[8:12]); got != 44 {
		t.Fatalf("the wave bank headerVersion is %d, want 44 (above 43, which changes what the parser reads)", got)
	}
	if got := binary.LittleEndian.Uint32(data[56:60]); got != 2 {
		t.Fatalf("the wave bank entryCount is %d, want 2", got)
	}
	dataOffset := binary.LittleEndian.Uint32(data[44:48])
	dataLength := binary.LittleEndian.Uint32(data[48:52])
	if dataOffset%4 != 0 {
		t.Fatalf("the wave data starts at %d, which is not 4-byte aligned as the bank's own alignment field declares", dataOffset)
	}
	if int(dataLength) != len(first)+len(second) {
		t.Fatalf("the declared wave data is %d bytes, want %d", dataLength, len(first)+len(second))
	}
	if int(dataOffset)+int(dataLength) != len(data) {
		t.Fatalf("the wave data runs to %d but the file is %d bytes", int(dataOffset)+int(dataLength), len(data))
	}
	// The declared bytes must be the wave bytes, in order. An offset that is
	// merely self-consistent would pass every check above and play silence.
	if !bytes.Equal(data[dataOffset:int(dataOffset)+len(first)], first) {
		t.Fatal("the first wave's declared region does not hold the first wave")
	}
	if !bytes.Equal(data[int(dataOffset)+len(first):], second) {
		t.Fatal("the second wave's declared region does not hold the second wave")
	}
}

func TestTheSoundBankFixturePointsEachCueAtTheSoundItNames(t *testing.T) {
	cues := []xactCue{
		{name: "Tone", waveIndex: 0, categoryIndex: 0},
		{name: "Blip", waveIndex: 1, categoryIndex: 1},
	}
	data := xactSoundBank("Sounds", "Waves", cues)
	if got := string(data[:4]); got != "SDBK" {
		t.Fatalf("the sound bank magic is %q, want SDBK", got)
	}
	if got := binary.LittleEndian.Uint16(data[19:21]); got != 2 {
		t.Fatalf("the sound bank cueSimpleCount is %d, want 2", got)
	}
	const headerSize = 138
	soundOffset := binary.LittleEndian.Uint32(data[70:74])
	if soundOffset != headerSize {
		t.Fatalf("the sound table starts at %d, want %d", soundOffset, headerSize)
	}
	cueSimpleOffset := binary.LittleEndian.Uint32(data[34:38])
	if cueSimpleOffset != headerSize+12*2 {
		t.Fatalf("the cue table starts at %d, want %d", cueSimpleOffset, headerSize+24)
	}
	// A simple cue's sound code is an ABSOLUTE file offset into the sound
	// table, which is the one field a plausible-looking layout gets wrong: a
	// RELATIVE index would be a small number that still parses.
	for index := range cues {
		entry := cueSimpleOffset + 5*uint32(index)
		code := binary.LittleEndian.Uint32(data[entry+1 : entry+5])
		want := soundOffset + 12*uint32(index)
		if code != want {
			t.Fatalf("cue %d points at %d, want the absolute sound offset %d", index, code, want)
		}
		if code < headerSize || code >= cueSimpleOffset {
			t.Fatalf("cue %d points at %d, which is outside the sound table", index, code)
		}
		// The sound it points at must carry the wave index the cue was
		// authored with.
		// A simple sound is flags(1) categoryIndex(2) volume(1) pitch(2)
		// priority(1) soundLength(2) waveIndex(2) waveBankIndex(1), so the wave
		// index sits at byte 9. Reading it a byte early answers 256 for a wave
		// index of 1, which is why this asserts the SECOND cue too: the first
		// one reads correctly at either offset because both its bytes are zero.
		if got := binary.LittleEndian.Uint16(data[code+9 : code+11]); got != cues[index].waveIndex {
			t.Fatalf("cue %d reaches a sound whose wave index is %d, want %d", index, got, cues[index].waveIndex)
		}
	}
	// The cue NAMES must be reachable at the offsets the name index declares,
	// because that is what SoundBank.GetCue looks a name up through.
	nameIndexOffset := binary.LittleEndian.Uint32(data[66:70])
	for index, cue := range cues {
		entry := nameIndexOffset + 6*uint32(index)
		nameOffset := binary.LittleEndian.Uint32(data[entry : entry+4])
		if int(nameOffset)+len(cue.name) > len(data) {
			t.Fatalf("cue %d's name offset %d is past the end of the file", index, nameOffset)
		}
		if got := string(data[nameOffset : int(nameOffset)+len(cue.name)]); got != cue.name {
			t.Fatalf("cue %d's name offset reaches %q, want %q", index, got, cue.name)
		}
	}
	if !bytes.Contains(data, append([]byte("Waves"), 0)) {
		t.Fatal("the sound bank does not name the wave bank it depends on")
	}
}

func TestTheSineGeneratorProducesPairedBytesAndFadesFromZero(t *testing.T) {
	wave := xactSine(440, 0.05)
	// 16-bit mono at 44100: the byte count is the sample count doubled, and an
	// ODD length would make the bank's declared play length disagree with the
	// format word's block alignment.
	if len(wave) != int(44100*0.05)*2 {
		t.Fatalf("the wave is %d bytes, want %d", len(wave), int(44100*0.05)*2)
	}
	if wave[0] != 0 || wave[1] != 0 {
		t.Fatalf("the wave does not start at silence: %#02x %#02x", wave[0], wave[1])
	}
}
