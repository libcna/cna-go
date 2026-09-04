package content

import (
	"reflect"
	"sync"
)

// ContentTypeReaderManager is
// Microsoft.Xna.Framework.Content.ContentTypeReaderManager:
//
//	.class public auto ansi sealed ContentTypeReaderManager
//
// The registry a content type reader uses to find the reader for a nested type.
//
// # Reference authority
//
//	Microsoft.Xna.Framework.dll   38e7093f52d7474b...
//
// # One public member, and it is a LOOKUP
//
// The type declares eleven methods and the pinned contract carries exactly one:
//
//	public ContentTypeReader GetTypeReader(Type targetType)
//
// whose whole body is thirteen bytes -- forward to a private static overload
// with the manager's own ContentReader. Everything else is `private` or
// `assembly`: ReadTypeManifest, InstantiateTypeReader, AddTypeReader,
// RollbackAddReaders, RollbackAddReader<T> and ContainsTypeReader.
//
// # That division is what keeps the type projectable
//
// The reference POPULATES its registry by reflection: InstantiateTypeReader
// takes the reader's type NAME as a string out of the .xnb manifest and builds
// it with Type.GetType. Go has no counterpart and never will.
//
// But none of that is contract surface. What a consumer can call is the lookup,
// and a lookup is a lookup. The population is CNA's job here --
// `cna_content_reader_initialize_type_readers` is what walks the manifest --
// which is the same division of labour the rest of the binding already has.
//
// # The three registries are STATIC and shared, and the reference locks them
//
//	private static Dictionary<string, ContentTypeReader> nameToReader
//	private static Dictionary<Type,   ContentTypeReader> targetTypeToReader
//	private static Dictionary<Type,   ContentTypeReader> readerTypeToReader
//
// Every lookup enters `Monitor.Enter(nameToReader)` -- one lock guarding all
// three. The projection keeps both properties: the registry is package-level
// rather than per-manager, and one mutex guards it. A reader registered while
// another goroutine looks one up is what the reference's lock exists for.
type ContentTypeReaderManager struct {
	// contentReader is the private `contentReader` field the public
	// GetTypeReader passes to the static overload. It is what a reader created
	// during the lookup would be initialised against.
	contentReader *ContentReader
}

// The process-wide registry, which is `private static` in the reference. It is
// package-level here for the same reason: a reader found for one manager is the
// same object every other manager finds.
var (
	typeReaderRegistryMutex sync.Mutex
	// targetTypeToReader is keyed by the type a reader DESERIALIZES.
	targetTypeToReader = map[reflect.Type]*ContentTypeReader{}
	// nameToReader is keyed by the reader's own CLR type name, which is what
	// the .xnb manifest carries. It is the lock's nominal subject in the
	// reference, and it is populated only by the assembly-visible path.
	nameToReader = map[string]*ContentTypeReader{}
)

// newContentTypeReaderManager builds the manager a ContentReader owns. It is
// unexported because the reference's constructor is `private`.
func newContentTypeReaderManager(reader *ContentReader) *ContentTypeReaderManager {
	return &ContentTypeReaderManager{contentReader: reader}
}

// GetTypeReader is ContentTypeReaderManager::GetTypeReader(Type), the type's
// one public member.
//
// The static overload it forwards to opens with
//
//	if (targetType == null) throw new ArgumentNullException("targetType");
//	lock (nameToReader) {
//	    if (targetTypeToReader.TryGetValue(targetType, out reader)) return reader;
//	    ...
//	}
//
// A type with no registered reader answers nil rather than failing: the
// reference's remaining path tries to instantiate one and returns null when it
// cannot, which is the branch a projection with no reflection always takes.
func (m *ContentTypeReaderManager) GetTypeReader(targetType reflect.Type) (*ContentTypeReader, error) {
	if targetType == nil {
		return nil, contentArgumentNullError("targetType")
	}
	typeReaderRegistryMutex.Lock()
	defer typeReaderRegistryMutex.Unlock()
	return targetTypeToReader[targetType], nil
}

// registerTypeReader is the assembly-visible AddTypeReader path, reduced to
// what a projection can do: record a reader under its target type and its
// reader-type name. It is unexported because every member that populates the
// registry in the reference is `private` or `assembly`.
func registerTypeReader(readerTypeName string, reader *ContentTypeReader) {
	if reader == nil {
		return
	}
	typeReaderRegistryMutex.Lock()
	defer typeReaderRegistryMutex.Unlock()
	if target := reader.TargetType(); target != nil {
		targetTypeToReader[target] = reader
	}
	if readerTypeName != "" {
		nameToReader[readerTypeName] = reader
	}
}

// containsTypeReader is the assembly-visible ContainsTypeReader.
func containsTypeReader(targetType reflect.Type) bool {
	if targetType == nil {
		return false
	}
	typeReaderRegistryMutex.Lock()
	defer typeReaderRegistryMutex.Unlock()
	_, present := targetTypeToReader[targetType]
	return present
}
