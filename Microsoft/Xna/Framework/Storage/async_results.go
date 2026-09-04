package storage

import (
	"sync"

	framework "github.com/openeggbert/cna-go/Microsoft/Xna/Framework"
)

// The Begin/End pairing, which is what the reference's private async-result
// classes carry as fields.
//
// # Why a registry rather than fields on AsyncResult
//
// The reference has two private result types -- StorageDeviceAsyncResult and
// StorageContainerOpenAsyncResult -- each holding what its End returns. CNA-Go
// projects `System.IAsyncResult` as ONE language adapter, because the contract
// names the interface and both concrete types are `private`; an adapter that
// carried a device field would be describing the wrong one half the time.
//
// So the pairing lives beside the adapter instead. That keeps AsyncResult
// exactly the four members IAsyncResult declares and nothing more, which is
// what the adapter registry pins.
//
// # Taking is what `endHasBeenCalled` is
//
// The reference's End sets a latch and refuses a second call:
//
//	if (typed.endHasBeenCalled)
//	    throw new InvalidOperationException(FrameworkResources.CannotEndTwice);
//	typed.endHasBeenCalled = true;
//
// Removing the entry is that latch. A second End finds nothing and answers the
// same refusal, and a result from a DIFFERENT Begin was never in the map, so
// the reference's two failures -- a wrong-typed result and a second End --
// collapse into the one a Go caller can actually reach.
var (
	asyncResultsMutex sync.Mutex
	selectorResults   = map[*framework.AsyncResult]*StorageDevice{}
	containerResults  = map[*framework.AsyncResult]*StorageContainer{}
)

func storeSelectorResult(result *framework.AsyncResult, device *StorageDevice) {
	asyncResultsMutex.Lock()
	defer asyncResultsMutex.Unlock()
	selectorResults[result] = device
}

func takeSelectorResult(result *framework.AsyncResult) (*StorageDevice, bool) {
	asyncResultsMutex.Lock()
	defer asyncResultsMutex.Unlock()
	device, present := selectorResults[result]
	if present {
		delete(selectorResults, result)
	}
	return device, present
}

func storeContainerResult(result *framework.AsyncResult, container *StorageContainer) {
	asyncResultsMutex.Lock()
	defer asyncResultsMutex.Unlock()
	containerResults[result] = container
}

func takeContainerResult(result *framework.AsyncResult) (*StorageContainer, bool) {
	asyncResultsMutex.Lock()
	defer asyncResultsMutex.Unlock()
	container, present := containerResults[result]
	if present {
		delete(containerResults, result)
	}
	return container, present
}
