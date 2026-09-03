package storage

import "testing"

// TestStorageDeviceNotConnectedExceptionIsAnExternalException pins the Storage
// package's share of the family: it derives from ExternalException, so it
// carries ErrorCode and the HResult-rendering ToString, and it is the second of
// the two types whose protected deserialization constructor is a recorded
// BLOCKED_DECLARED_MEMBER.
func TestStorageDeviceNotConnectedExceptionIsAnExternalException(t *testing.T) {
	const name = "Microsoft.Xna.Framework.Storage.StorageDeviceNotConnectedException"
	exception := NewStorageDeviceNotConnectedExceptionByNone()
	if got := exception.ErrorCode(); got != -2147467259 {
		t.Fatalf("ErrorCode = %d, want E_FAIL", got)
	}
	if got := exception.Message(); got != "External component has thrown an exception." {
		t.Fatalf("default Message = %q", got)
	}
	if got := exception.ToString(); got != name+" (0x80004005): External component has thrown an exception." {
		t.Fatalf("ToString = %q", got)
	}
	withMessage := NewStorageDeviceNotConnectedExceptionByString("no storage device")
	if got := withMessage.ToString(); got != name+" (0x80004005): no storage device" {
		t.Fatalf("ToString with a message = %q", got)
	}
}
