package content

import "errors"

// errContentLoad projects the
// Microsoft.Xna.Framework.Content.ContentLoadException the pipeline throws. It
// is a Go sentinel rather than the projected ContentLoadException TYPE for the
// reason every other family's refusals are: the projected exception carries the
// CLR surface a consumer inspects, and a Go `error` is what a Go caller tests.
// The two are different jobs and the project has kept them apart since
// Foundation 76.
var errContentLoad = errors.New("content could not be loaded")

// The FrameworkResources messages the content pipeline formats, verified byte
// for byte against the retained assembly. Each carries the asset name first,
// which is what makes a load failure name the file that caused it.
const (
	// badXnbWrongType names what the file contains and then what the load asked
	// for, in that order.
	badXnbWrongType = "Error loading \"%s\". File contains %v but trying to load as %v."
	// badXnbSize is the truncation check, which PrepareStream performs only
	// when the stream can seek.
	badXnbSize = "Error loading \"%s\". File has been truncated."
	// badXnbMagic is the four-byte header check.
	badXnbMagic = "Error loading \"%s\". This is not a compiled content file."
	// badXnbVersion is the format-version check.
	badXnbVersion = "Error loading \"%s\". This file was compiled using the wrong version of the XNA Framework."
)
