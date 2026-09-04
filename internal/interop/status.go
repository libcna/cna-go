package interop

import (
	"errors"
	"fmt"
)

const (
	resultSuccess         uint32 = 0
	resultInvalidArgument uint32 = 1
	resultInvalidHandle   uint32 = 2
	resultInvalidState    uint32 = 3
	resultThread          uint32 = 8
	resultCallback        uint32 = 9
)

var (
	ErrNativeUnavailable = errors.New("CNA native C ABI is unavailable")
	ErrWrongThread       = errors.New("CNA operation must run on the Game owner OS thread")
	ErrStaleGeneration   = errors.New("CNA resource belongs to an inactive Game generation")
	ErrOutsideCallback   = errors.New("CNA graphics/input operation requires an active Game lifecycle callback")
	ErrDisposed          = errors.New("CNA resource is disposed")
	ErrChildrenAlive     = errors.New("CNA parent resource still has live children")
)

// NativeError retains the stable CNA_Result code without exposing a C type.
type NativeError struct {
	Operation string
	Code      uint32
	Message   string
}

func (e *NativeError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("%s failed with CNA result %d", e.Operation, e.Code)
	}
	return fmt.Sprintf("%s failed with CNA result %d: %s", e.Operation, e.Code, e.Message)
}

// cnaResultBufferTooSmall is CNA_RESULT_BUFFER_TOO_SMALL, which the canonical
// abi.h defines as 14. It is named here because ONE bound route uses it as a
// success: cna_title_container_read_ext sizes and copies through the same
// entry point, and a zero capacity answers this code with the byte count
// filled in.
const cnaResultBufferTooSmall uint32 = 14

func resultError(operation string, code uint32) error {
	if code == resultSuccess {
		return nil
	}
	if code == resultThread {
		return fmt.Errorf("%s: %w", operation, ErrWrongThread)
	}
	return &NativeError{Operation: operation, Code: code, Message: nativeLastErrorMessage()}
}
