package media

import (
	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
)

// VisualizationData is the pair of fixed-size buffers XNA's media playback
// pipeline fills with frequency and sample data.
//
// Microsoft.Xna.Framework.dll shows the whole type as four fields and two one
// -ldfld getters. The constructor is
//
//	newarr [mscorlib]System.Single  (0x100)  -> frequencies
//	newarr [mscorlib]System.Single  (0x100)  -> samples
//	newobj ReadOnlyCollection`1<float32>(IList`1<!0>)  over each
//
// so it allocates two 256-element Single arrays and wraps each in a read-only
// view. It validates nothing, reaches no device, and touches no native code.
//
// The two arrays are `assembly` fields, not public surface: they are what the
// media backend writes into. The public views are LIVE over them, because
// ReadOnlyCollection<T> stores the list it is given rather than copying it, so
// a caller holding Frequencies sees each buffer refresh without asking again.
// Read-only means the caller cannot write through the view, not that the data
// is frozen.
//
// Completing this type starts no playback. CNA-Go has no MediaPlayer, no Song,
// and no media backend, so nothing ever writes into the buffers and both views
// stay 256 zeros for their whole lifetime.
type VisualizationData struct {
	frequencies []float32
	samples     []float32

	frequenciesCollection *framework.ReadOnlyCollection[float32]
	samplesCollection     *framework.ReadOnlyCollection[float32]
}

// visualizationBufferLength is the 0x100 the reference constructor allocates
// for both buffers.
const visualizationBufferLength = 0x100

// NewVisualizationData allocates both 256-element buffers and their views,
// exactly as the reference constructor does. It validates nothing and cannot
// fail.
func NewVisualizationData() *VisualizationData {
	data := &VisualizationData{
		frequencies: make([]float32, visualizationBufferLength),
		samples:     make([]float32, visualizationBufferLength),
	}
	data.frequenciesCollection = framework.NewReadOnlyCollectionOverSingles(data.frequencies)
	data.samplesCollection = framework.NewReadOnlyCollectionOverSingles(data.samples)
	return data
}

// Frequencies is the live read-only view over the frequency buffer. The getter
// is one ldfld and cannot fail.
func (d *VisualizationData) Frequencies() *framework.ReadOnlyCollection[float32] {
	return d.frequenciesCollection
}

// Samples is the live read-only view over the sample buffer. The getter is one
// ldfld and cannot fail.
func (d *VisualizationData) Samples() *framework.ReadOnlyCollection[float32] {
	return d.samplesCollection
}
