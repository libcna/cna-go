package interop

import "errors"

// ErrNativeUnavailable reports that the canonical CNA C ABI is unavailable.
var ErrNativeUnavailable = errors.New("CNA native C ABI is not available yet")
