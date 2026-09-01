package graphics

import (
	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
)

// ---------------------------------------------------------------------------
// Foundation 68 — DisplayModeCollection.
// ---------------------------------------------------------------------------

// DisplayModeCollection is
// Microsoft.Xna.Framework.Graphics.DisplayModeCollection:
//
//	.class public auto ansi beforefieldinit DisplayModeCollection
//	       extends System.Object
//	       implements IEnumerable`1<DisplayMode>, IEnumerable
//
// Two public members over one private `List<DisplayMode>`. Its constructor is
// `assembly` -- a consumer cannot build one in C# either -- so CNA-Go declares
// no exported constructor and the collection is reached only through
// GraphicsAdapter::SupportedDisplayModes.
//
// # The indexer FILTERS, it does not index
//
//	get_Item(SurfaceFormat format)
//	  List<DisplayMode> matches = new List<DisplayMode>();
//	  foreach (DisplayMode mode in _displayModes)
//	      if (mode.Format == format) matches.Add(mode);
//	  return matches;
//
// It is typed `IEnumerable<DisplayMode>` rather than `DisplayMode`, and its
// argument is a FORMAT rather than a position. A reader who saw "indexer" and
// expected `modes[3]` would be wrong about both halves. It also allocates a
// fresh list per call, so two calls never share one.
//
// Both members are infallible: the reference reads a managed list and copies.
type DisplayModeCollection struct {
	// displayModes is `_displayModes`, in the order CNA reported them.
	displayModes []*DisplayMode
}

// newDisplayModeCollection is the `assembly` constructor. It COPIES, because
// the caller's slice is CNA's readback buffer and the collection outlives it.
func newDisplayModeCollection(modes []*DisplayMode) *DisplayModeCollection {
	return &DisplayModeCollection{displayModes: append([]*DisplayMode(nil), modes...)}
}

// Item is DisplayModeCollection::get_Item(SurfaceFormat), the filter above.
//
// Its CLR type is `IEnumerable<DisplayMode>`, and the settled BCL-interface
// rule projects a BCL interface to no Go interface -- so the position is `any`,
// exactly as `System.IServiceProvider` is. What it HOLDS is the language
// adapter for a sequence in this profile, `framework.Iterator[*DisplayMode]`,
// which is what GetEnumerator returns for the same elements.
//
// A format nothing matches answers a sequence that terminates immediately,
// which is what the reference's fresh empty list does. Whether the slice behind
// it is nil or empty is NOT observable through the iterator, so no claim is
// made about it.
func (c *DisplayModeCollection) Item(format SurfaceFormat) any {
	matches := make([]*DisplayMode, 0)
	if c != nil {
		for _, mode := range c.displayModes {
			if mode.Format() == format {
				matches = append(matches, mode)
			}
		}
	}
	return &displayModeIterator{modes: matches}
}

// GetEnumerator is DisplayModeCollection::GetEnumerator, which forwards to the
// private list's own enumerator -- so it walks every mode, in order, unfiltered.
func (c *DisplayModeCollection) GetEnumerator() framework.Iterator[*DisplayMode] {
	if c == nil {
		return &displayModeIterator{}
	}
	return &displayModeIterator{modes: c.displayModes}
}

// displayModeIterator is the Go language adapter for the List<DisplayMode>
// enumerator both members hand out. The list it walks is never mutated after
// the collection is built -- CNA reports the modes once -- so there is no
// version check to reproduce and no enumeration failure to report.
type displayModeIterator struct {
	modes []*DisplayMode
	at    int
}

func (i *displayModeIterator) Next() (*DisplayMode, bool, error) {
	if i == nil || i.at >= len(i.modes) {
		return nil, false, nil
	}
	mode := i.modes[i.at]
	i.at++
	return mode, true, nil
}
