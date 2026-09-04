package content

import (
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"
)

// TestContentTypeReaderIsPureFieldAccess pins the four members and the one
// measured quirk of the constructor.
func TestContentTypeReaderIsPureFieldAccess(t *testing.T) {
	reader := NewContentTypeReader(reflect.TypeOf(int32(0)))
	if reader.TargetType() != reflect.TypeOf(int32(0)) {
		t.Fatalf("TargetType = %v", reader.TargetType())
	}
	// Both public virtuals have `ldc.i4.0; ret` base bodies.
	if reader.TypeVersion() != 0 {
		t.Fatalf("TypeVersion = %d, want 0", reader.TypeVersion())
	}
	if reader.CanDeserializeIntoExistingObject() {
		t.Fatal("CanDeserializeIntoExistingObject was true; the base body is ldc.i4.0")
	}
	// Initialize's base body is ONE `ret`.
	if err := reader.Initialize(nil); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	// A NULL target type is accepted -- the reference guards the
	// TargetIsValueType assignment with op_Inequality rather than refusing.
	bare := NewContentTypeReader(nil)
	if bare.TargetType() != nil {
		t.Fatal("a nil target type was replaced")
	}
	if bare.targetIsValue() {
		t.Fatal("a nil target type set TargetIsValueType; the reference guards that assignment")
	}
}

// TestTargetIsValueTypeFollowsTheGoKind pins the System.Type::get_IsValueType
// projection: a CLR class projects to a Go pointer and a CLR struct to a value,
// which is what makes the correspondence exact.
func TestTargetIsValueTypeFollowsTheGoKind(t *testing.T) {
	for _, value := range []any{int32(0), float32(0), struct{ X int }{}, [4]byte{}, true} {
		if !NewContentTypeReader(reflect.TypeOf(value)).targetIsValue() {
			t.Fatalf("%T was not treated as a CLR value type", value)
		}
	}
	for _, value := range []any{&struct{}{}, []int{}, map[string]int{}, io.Reader(nil)} {
		typ := reflect.TypeOf(value)
		if typ == nil {
			continue
		}
		if NewContentTypeReader(typ).targetIsValue() {
			t.Fatalf("%T was treated as a CLR value type", value)
		}
	}
}

// TestContentTypeReaderReadRefuses pins that the abstract member has no body to
// run and says which type it was built for.
func TestContentTypeReaderReadRefuses(t *testing.T) {
	reader := NewContentTypeReader(reflect.TypeOf(int32(0)))
	_, err := reader.Read(nil, nil)
	if !errors.Is(err, errContentLoad) {
		t.Fatalf("Read = %v, want a content-load refusal", err)
	}
	if !contains(err.Error(), "int32") {
		t.Fatalf("the refusal did not name the target type: %q", err.Error())
	}
	// A reader over no type still names something rather than panicking.
	if _, err := NewContentTypeReader(nil).Read(nil, nil); err == nil {
		t.Fatal("a reader over no type accepted Read")
	}
}

// TestContentTypeReaderOfTTakesItsTypeFromT pins `base..ctor(typeof(T))`.
func TestContentTypeReaderOfTTakesItsTypeFromT(t *testing.T) {
	reader := NewContentTypeReaderOfT[int32]()
	if reader.TargetType() != reflect.TypeOf(int32(0)) {
		t.Fatalf("TargetType = %v, want int32", reader.TargetType())
	}
	pointer := NewContentTypeReaderOfT[*ContentManager]()
	if pointer.TargetType() != reflect.TypeOf((*ContentManager)(nil)) {
		t.Fatalf("TargetType = %v", pointer.TargetType())
	}
	// The inherited getters answer the base's defaults.
	if reader.TypeVersion() != 0 || reader.CanDeserializeIntoExistingObject() {
		t.Fatal("the generic reader did not inherit the base's defaults")
	}
}

// TestContentTypeReaderOfTCastRefusesTheWrongInstance pins the measured cast:
// a null existing instance becomes default(T), a WRONG one is the error, and
// the message names BOTH types.
func TestContentTypeReaderOfTCastRefusesTheWrongInstance(t *testing.T) {
	reader := NewContentTypeReaderOfT[int32]()
	// A wrong existing instance fails the cast before anything else.
	_, err := reader.ReadByContentReaderAndObject(nil, "not an int32")
	if !errors.Is(err, errContentLoad) {
		t.Fatalf("a wrong instance = %v, want a content-load refusal", err)
	}
	if !contains(err.Error(), "string") {
		t.Fatalf("the refusal did not name what the file contains: %q", err.Error())
	}
	// A NIL existing instance is default(T) rather than a refusal, so it gets
	// past the cast and reaches the abstract member's own refusal.
	_, err = reader.ReadByContentReaderAndObject(nil, nil)
	if !errors.Is(err, errContentLoad) {
		t.Fatalf("a nil instance = %v", err)
	}
	if contains(err.Error(), "string") {
		t.Fatal("a nil instance took the wrong-type path; the reference makes it default(T)")
	}
}

// TestContentTypeReaderManagerLookupRefusesNil pins the one public member.
func TestContentTypeReaderManagerLookupRefusesNil(t *testing.T) {
	manager := newContentTypeReaderManager(nil)
	if _, err := manager.GetTypeReader(nil); !errors.Is(err, errContentArgumentNull) {
		t.Fatalf("GetTypeReader(nil) = %v, want the argument-null refusal", err)
	}
	// A type with no registered reader answers nil and no error: the
	// reference's remaining path tries to instantiate one and returns null when
	// it cannot, which is the branch a projection with no reflection takes.
	reader, err := manager.GetTypeReader(reflect.TypeOf(int32(0)))
	if err != nil {
		t.Fatalf("GetTypeReader: %v", err)
	}
	if reader != nil {
		t.Fatal("a type with no registered reader answered one")
	}
	// A registered reader is found, and the registry is process-wide as the
	// reference's three static dictionaries are.
	registered := NewContentTypeReader(reflect.TypeOf(uint16(0)))
	registerTypeReader("StressReader", registered)
	found, err := manager.GetTypeReader(reflect.TypeOf(uint16(0)))
	if err != nil || found != registered {
		t.Fatalf("GetTypeReader = %v, %v", found, err)
	}
	if !containsTypeReader(reflect.TypeOf(uint16(0))) {
		t.Fatal("ContainsTypeReader missed a registered reader")
	}
}

// TestResourceContentManagerRefusals pins the constructor's guard order and
// OpenStream's two DIFFERENT messages.
func TestResourceContentManagerRefusals(t *testing.T) {
	// The BASE constructor runs first, so a nil service provider is refused
	// before the resource manager is even looked at.
	if _, err := NewResourceContentManager(nil, nil); err == nil {
		t.Fatal("a nil service provider was accepted")
	}
	if _, err := NewResourceContentManager(struct{}{}, nil); !errors.Is(err, errContentArgumentNull) {
		t.Fatalf("a nil resource manager = %v, want the argument-null refusal", err)
	}

	payload := []byte("asset bytes")
	manager, err := NewResourceContentManager(struct{}{}, func(name string) any {
		switch name {
		case "binary":
			return payload
		case "text":
			return "not a byte array"
		}
		return nil
	})
	if err != nil {
		t.Fatalf("NewResourceContentManager: %v", err)
	}

	stream, err := manager.OpenStream("binary")
	if err != nil {
		t.Fatalf("OpenStream: %v", err)
	}
	read, err := io.ReadAll(stream)
	if err != nil || string(read) != string(payload) {
		t.Fatalf("read back %q, %v", read, err)
	}

	// The two refusals carry DIFFERENT messages: absent versus not binary.
	_, err = manager.OpenStream("missing")
	if err == nil || !contains(err.Error(), "Resource not found") {
		t.Fatalf("a missing resource = %v, want the not-found message", err)
	}
	_, err = manager.OpenStream("text")
	if err == nil || !contains(err.Error(), "Not a binary resource") {
		t.Fatalf("a non-binary resource = %v, want the not-binary message", err)
	}
	if contains(err.Error(), "Resource not found") {
		t.Fatal("the two refusals were collapsed into one message")
	}
}

// TestResourceContentManagerNamesItselfWhenDisposed is the identity site: the
// base raises ObjectDisposedException(this.ToString()), so a disposed
// ResourceContentManager must name ResourceContentManager.
func TestResourceContentManagerNamesItselfWhenDisposed(t *testing.T) {
	manager, err := NewResourceContentManager(struct{}{}, func(string) any { return nil })
	if err != nil {
		t.Fatalf("NewResourceContentManager: %v", err)
	}
	if err := manager.Dispose(); err != nil {
		t.Fatalf("Dispose: %v", err)
	}
	err = manager.Unload()
	if err == nil {
		t.Fatal("a disposed manager accepted Unload")
	}
	if !contains(err.Error(), "ResourceContentManager") {
		t.Fatalf("the refusal named %q; the CLR `this` is the DERIVED object", err.Error())
	}

	// A bare ContentManager names itself, which is the other half of the claim.
	bare, err := NewContentManagerByIServiceProvider(struct{}{})
	if err != nil {
		t.Fatalf("NewContentManager: %v", err)
	}
	if err := bare.DisposeByNone(); err != nil {
		t.Fatalf("Dispose: %v", err)
	}
	err = bare.Unload()
	if err == nil || contains(err.Error(), "ResourceContentManager") {
		t.Fatalf("a bare manager named %v", err)
	}
	if !contains(err.Error(), "Content.ContentManager") {
		t.Fatalf("a bare manager named %q", err.Error())
	}
}

// TestResourceContentManagerReachesTheGenericFunctions pins that the
// substitutable base interface does what it exists for: a derived manager can
// be handed to the package functions.
func TestResourceContentManagerReachesTheGenericFunctions(t *testing.T) {
	manager, err := NewResourceContentManager(struct{}{}, func(string) any { return nil })
	if err != nil {
		t.Fatalf("NewResourceContentManager: %v", err)
	}
	var _ ContentManagerReference = manager
	var _ ContentManagerReference = manager.contentManager()
	// It reaches the function; the load itself refuses because the asset type
	// is outside the closed set, which is a DIFFERENT refusal from a type
	// error and is what proves the call happened.
	if _, err := ContentManagerLoad[*struct{}](manager, "asset"); err == nil {
		t.Fatal("the load succeeded for an unsupported asset type")
	}
	// A nil reference is the CLR's null and answers the disposal refusal
	// rather than panicking.
	if _, err := ContentManagerLoad[*struct{}](nil, "asset"); err == nil {
		t.Fatal("a nil reference was accepted")
	}
}

func contains(haystack, needle string) bool {
	if len(needle) > len(haystack) {
		return false
	}
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}

// TestContentTypeReaderCtorAcceptsANilTargetType pins that the constructor is
// TOTAL. The reference's `.ctor(Type)` stores the argument and calls
// get_IsValueType on it; a null would throw there, but the CLR's own
// `typeof(T)` can never be null, so no reader in the reference reaches it.
//
// Go can pass a nil reflect.Type, and calling Kind() on one panics. The
// projection guards the get_IsValueType call rather than refusing the
// construction, because refusing would be a behaviour the reference does not
// have -- and a reader built this way is still usable, it simply answers false
// to TargetIsValueType.
func TestContentTypeReaderCtorAcceptsANilTargetType(t *testing.T) {
	reader := NewContentTypeReader(nil)
	if reader == nil {
		t.Fatal("NewContentTypeReader(nil) refused; the reference's constructor has no such refusal")
	}
	if reader.TargetType() != nil {
		t.Fatal("the nil target type was not stored")
	}
	if reader.targetIsValueType {
		t.Fatal("a nil target type was called a value type")
	}
	// Every other member still answers.
	if reader.TypeVersion() != 0 || reader.CanDeserializeIntoExistingObject() {
		t.Fatal("the virtuals stopped answering their defaults")
	}
	if err := reader.Initialize(nil); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
}

// TestContentTypeReaderOfTAcceptsANilExistingInstance is the `existingInstance
// == null` arm of the measured Read: it takes default(T) WITHOUT running the
// type check. Running the check anyway would refuse every FRESH read, because a
// nil interface is not a T.
//
// Both arms end in an error here, because the typed Read they hand off to is
// abstract and has no body in the reference either. What separates them is
// WHICH error: the nil arm must reach the abstract refusal, and only the wrong
// arm may raise BadXnbWrongType.
func TestContentTypeReaderOfTAcceptsANilExistingInstance(t *testing.T) {
	reader := NewContentTypeReaderOfT[int32]()

	_, err := reader.ReadByContentReaderAndObject(nil, nil)
	if err == nil {
		t.Fatal("the abstract typed Read answered")
	}
	if strings.Contains(err.Error(), "but trying to load as") {
		t.Fatalf("a nil existing instance raised the wrong-type refusal: %v", err)
	}
	if !strings.Contains(err.Error(), "no content type reader is available") {
		t.Fatalf("the nil arm did not reach the abstract Read: %v", err)
	}

	// The wrong arm, for contrast, does raise it.
	if _, err = reader.ReadByContentReaderAndObject(nil, "not an int32"); err == nil {
		t.Fatal("a wrong existing instance was accepted")
	} else if !strings.Contains(err.Error(), "but trying to load as") {
		t.Fatalf("the wrong arm raised %q; want the BadXnbWrongType message", err)
	}

	// A pointer T is the case where default(T) is a nil pointer rather than a
	// zero value, and the nil arm must still not run the check.
	pointers := NewContentTypeReaderOfT[*ContentReader]()
	if _, err = pointers.ReadByContentReaderAndObject(nil, nil); err == nil {
		t.Fatal("the abstract typed Read answered for a pointer T")
	} else if strings.Contains(err.Error(), "but trying to load as") {
		t.Fatalf("a nil existing instance raised the wrong-type refusal for a pointer T: %v", err)
	}
}

// TestResourceContentManagerChecksTheServiceProviderFirst pins the ORDER of the
// two refusals. The reference's constructor is
//
//	: base(serviceProvider)
//	{ if (resourceManager == null) throw new ArgumentNullException(...); }
//
// so the base constructor -- and its own serviceProvider check -- runs BEFORE
// the resourceManager check. With both arguments missing, the message names
// serviceProvider.
func TestResourceContentManagerChecksTheServiceProviderFirst(t *testing.T) {
	_, err := NewResourceContentManager(nil, nil)
	if err == nil {
		t.Fatal("two missing arguments were accepted")
	}
	if !strings.Contains(err.Error(), "serviceProvider") {
		t.Fatalf("the refusal was %q; the base constructor runs first, so it must name serviceProvider", err)
	}
	if strings.Contains(err.Error(), "resourceManager") {
		t.Fatalf("the refusal was %q; the resourceManager check is not reached", err)
	}
}

// TestReadAssetsDisposalRefusalNamesTheRuntimeType pins ReadAsset's OWN
// identity site -- the third of the three. The reference gives ReadAsset its
// own disposal check rather than letting it inherit Load's, so a disposed
// ResourceContentManager reached through ReadAsset names itself.
//
// What this test deliberately does NOT claim: that ReadAsset forwards the
// substitutable REFERENCE to Load rather than the narrowed base. It forwards
// the reference, and that is the faithful spelling -- the CLR `this` does not
// change across a call -- but it is not observable today, because ReadAsset's
// own guard fires before the delegation and Load's only identity site is the
// same disposal check. A test asserting the forwarding passes with the
// narrowing planted, so it would be measuring this member's guard while
// appearing to measure the call.
func TestReadAssetsDisposalRefusalNamesTheRuntimeType(t *testing.T) {
	manager, err := NewResourceContentManager(struct{}{}, func(string) any { return nil })
	if err != nil {
		t.Fatalf("NewResourceContentManager: %v", err)
	}
	if err = manager.Dispose(); err != nil {
		t.Fatalf("Dispose: %v", err)
	}

	_, err = ContentManagerReadAsset[*struct{}](manager, "asset", nil)
	if err == nil {
		t.Fatal("a disposed manager read an asset")
	}
	if !strings.Contains(err.Error(), "ResourceContentManager") {
		t.Fatalf("ReadAsset refused with %q; the CLR `this` is the derived object, so it must name ResourceContentManager", err)
	}

	// Load reached directly is the second site and says the same thing.
	if _, err = ContentManagerLoad[*struct{}](manager, "asset"); err == nil {
		t.Fatal("a disposed manager loaded an asset")
	} else if !strings.Contains(err.Error(), "ResourceContentManager") {
		t.Fatalf("Load refused with %q; want the derived type named", err)
	}

	// And the base manager names ITSELF, so the two assertions above are about
	// the identity rather than about the message always saying "Resource".
	base, err := NewContentManagerByIServiceProvider(struct{}{})
	if err != nil {
		t.Fatalf("NewContentManagerByIServiceProvider: %v", err)
	}
	if err = base.DisposeByNone(); err != nil {
		t.Fatalf("DisposeByNone: %v", err)
	}
	if _, err = ContentManagerReadAsset[*struct{}](base, "asset", nil); err == nil {
		t.Fatal("a disposed base manager read an asset")
	} else if strings.Contains(err.Error(), "ResourceContentManager") {
		t.Fatalf("the base manager refused with %q; it must name its own type", err)
	}
}
