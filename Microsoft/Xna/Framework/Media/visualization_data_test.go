package media

import "testing"

func TestNewVisualizationDataAllocatesTwoFullBuffers(t *testing.T) {
	data := NewVisualizationData()
	if data.Frequencies().Count() != 0x100 || data.Samples().Count() != 0x100 {
		t.Fatalf("counts = %d/%d, want 256/256",
			data.Frequencies().Count(), data.Samples().Count())
	}
	// The constructor allocates and wraps; it writes nothing, so both buffers
	// start as 256 zeros.
	for index := int32(0); index < 0x100; index++ {
		frequency, err := data.Frequencies().Item(index)
		if err != nil {
			t.Fatal(err)
		}
		sample, err := data.Samples().Item(index)
		if err != nil {
			t.Fatal(err)
		}
		if frequency != 0 || sample != 0 {
			t.Fatalf("index %d = %v/%v, want 0/0", index, frequency, sample)
		}
	}
}

func TestVisualizationDataViewsAreStableAndDistinct(t *testing.T) {
	data := NewVisualizationData()
	// Both getters are one ldfld over a field the constructor set, so each
	// returns the same view every time.
	if data.Frequencies() != data.Frequencies() || data.Samples() != data.Samples() {
		t.Fatal("a getter must return the same stored view")
	}
	if data.Frequencies() == data.Samples() {
		t.Fatal("the two buffers must have distinct views")
	}
	// Two instances share nothing.
	other := NewVisualizationData()
	if other.Frequencies() == data.Frequencies() {
		t.Fatal("two instances shared a view")
	}
}

func TestVisualizationDataViewsAreReadOnlyButLive(t *testing.T) {
	data := NewVisualizationData()
	// A caller cannot write through the view: the CLR type's mutators are all
	// private explicit implementations, so none is projected. What a caller
	// can do is copy out.
	destination := make([]float32, 0x100)
	if err := data.Frequencies().CopyTo(destination, 0); err != nil {
		t.Fatal(err)
	}
	destination[0] = 1
	first, err := data.Frequencies().Item(0)
	if err != nil {
		t.Fatal(err)
	}
	if first != 0 {
		t.Fatal("CopyTo aliased the buffer")
	}

	// The view is live over the buffer the backend writes into. CNA-Go has no
	// media backend, so this test plays the backend's part directly on the
	// unexported field, which is exactly what the assembly-visible reference
	// field exists for.
	data.frequencies[5] = 0.25
	updated, err := data.Frequencies().Item(5)
	if err != nil {
		t.Fatal(err)
	}
	if updated != 0.25 {
		t.Fatalf("Item(5) = %v, want 0.25: the view must be live over the buffer", updated)
	}
	if !data.Frequencies().Contains(0.25) {
		t.Fatal("a search must see the current contents")
	}
}

func TestVisualizationDataEnumeratesTheWholeBuffer(t *testing.T) {
	data := NewVisualizationData()
	data.samples[0] = 1
	data.samples[0xFF] = -1
	iterator := data.Samples().GetEnumerator()
	seen, sum := 0, float32(0)
	for {
		value, ok, err := iterator.Next()
		if err != nil {
			t.Fatal(err)
		}
		if !ok {
			break
		}
		seen++
		sum += value
	}
	if seen != 0x100 {
		t.Fatalf("enumerated %d elements, want 256", seen)
	}
	if sum != 0 {
		t.Fatalf("sum = %v, want 0", sum)
	}
}
