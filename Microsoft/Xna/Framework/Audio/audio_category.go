package audio

import (
	"github.com/openeggbert/cna-go/internal/interop"
)

// AudioCategory is Microsoft.Xna.Framework.Audio.AudioCategory:
//
//	.class public sequential ansi sealed beforefieldinit AudioCategory
//	       extends [mscorlib]System.ValueType
//	       implements [mscorlib]System.IEquatable`1<AudioCategory>
//
// One authored category from the engine's settings file -- "Music", "SFX" --
// through which a game pauses, resumes, stops or re-levels every cue assigned
// to it at once.
//
// # Reference authority
//
//	Microsoft.Xna.Framework.Xact.dll   a14d5364dca7cf49...
//
// # It is a VALUE TYPE holding a handle, which is the whole difficulty
//
// The reference's struct holds an engine reference and an INDEX, so copying it
// is free and two copies are equal because their indices are. CNA has no index:
// every `GetCategory` returns a fresh OWNED handle. So the projection's value
// holds the handle, and three consequences follow, each of which decided a
// member below.
//
// Copying is still free, and two copies of one value share a handle -- which is
// correct, because they denote one category.
//
// Two SEPARATE lookups of one name produce two handles that must compare EQUAL.
// That is what cna_audio_category_equals answers, and it is why equality here
// reaches a runtime where the reference's compares two fields.
//
// Nothing here destroys the handle. A struct has no Dispose, and the reference
// declares none, so inventing one would add a member the contract does not have.
// The handle is registered as a child of the ENGINE instead, and the engine's
// disposal releases it -- which is the only lifetime a value type can have.
//
// # The ZERO value is legal and every member must survive it
//
// `default(AudioCategory)` is reachable in the reference and reaches a null
// engine reference, where the reference's own members throw
// NullReferenceException. Go cannot project that, so each member below answers
// the nil-resource refusal in its own words.
type AudioCategory struct {
	// resource is the owned CNA handle, or nil in the zero value.
	resource *interop.Resource
}

// Name is AudioCategory::get_Name.
//
// The reference reads the authored name out of the engine's table. This reaches
// CNA because the projection holds no table: the handle is the only thing that
// knows which category it is.
func (c AudioCategory) Name() (string, error) {
	if c.resource == nil {
		return "", errXactNil
	}
	return c.resource.AudioCategoryName()
}

// SetVolume is AudioCategory::SetVolume(Single), which scales every cue in the
// category.
func (c AudioCategory) SetVolume(volume float32) error {
	if c.resource == nil {
		return errXactNil
	}
	return c.resource.AudioCategorySetVolume(volume)
}

// Pause is AudioCategory::Pause().
func (c AudioCategory) Pause() error {
	if c.resource == nil {
		return errXactNil
	}
	return c.resource.AudioCategoryPause()
}

// Resume is AudioCategory::Resume().
func (c AudioCategory) Resume() error {
	if c.resource == nil {
		return errXactNil
	}
	return c.resource.AudioCategoryResume()
}

// Stop is AudioCategory::Stop(AudioStopOptions).
//
// The options word passes through unvalidated. AudioStopOptions is not a
// `[Flags]` enum -- the pinned contract's `flags` is false -- so its two values
// are alternatives rather than a set, and which combinations mean anything is
// XACT's decision, not this projection's.
func (c AudioCategory) Stop(options AudioStopOptions) error {
	if c.resource == nil {
		return errXactNil
	}
	return c.resource.AudioCategoryStop(uint32(options))
}

// GetHashCode is AudioCategory::GetHashCode.
//
// The reference hashes the authored NAME, so two lookups of one name hash the
// same. This reaches CNA for the same reason equality does: the projection
// holds no name to hash, and hashing the name it would have to fetch first
// would be a second answer that could disagree with equality's.
//
// It is therefore FALLIBLE where the reference's is not, which is the settled
// shape for a hash that reaches a runtime -- the runtimeReadMembers registry
// exists for exactly this.
func (c AudioCategory) GetHashCode() (int32, error) {
	if c.resource == nil {
		return 0, errXactNil
	}
	return c.resource.AudioCategoryHashCode()
}

// EqualsByAudioCategory is AudioCategory::Equals(AudioCategory), the
// IEquatable<AudioCategory> member.
//
// Two ZERO values are equal, and a zero value equals nothing else. That is what
// the reference gives -- two default structs hold the same null engine and the
// same zero index -- and it is decided here before either handle is touched,
// because a nil resource has nothing to ask.
func (c AudioCategory) EqualsByAudioCategory(other AudioCategory) (bool, error) {
	if c.resource == nil || other.resource == nil {
		return c.resource == other.resource, nil
	}
	return c.resource.AudioCategoryEquals(other.resource)
}

// EqualsByObject is AudioCategory::Equals(Object).
//
// The reference's is `obj is AudioCategory && Equals((AudioCategory)obj)`, so a
// value of any other type -- and a null -- answers false rather than throwing.
// Go's two-value type assertion has exactly that behaviour.
func (c AudioCategory) EqualsByObject(obj any) (bool, error) {
	other, ok := obj.(AudioCategory)
	if !ok {
		return false, nil
	}
	return c.EqualsByAudioCategory(other)
}

// ToString is AudioCategory::ToString(), which answers the authored NAME.
//
// This is the one member where the reference's ToString carries information --
// RendererDetail's answered its type name and nothing else -- so it forwards to
// Name rather than to a constant.
func (c AudioCategory) ToString() (string, error) {
	return c.Name()
}

// AudioCategoryOperatorEqualityByAudioCategoryAndAudioCategory is
// AudioCategory::op_Equality, which the reference implements as
// `value1.Equals(value2)`.
func AudioCategoryOperatorEqualityByAudioCategoryAndAudioCategory(value1, value2 AudioCategory) (bool, error) {
	return value1.EqualsByAudioCategory(value2)
}

// AudioCategoryOperatorInequalityByAudioCategoryAndAudioCategory is
// AudioCategory::op_Inequality, `!op_Equality(value1, value2)`.
func AudioCategoryOperatorInequalityByAudioCategoryAndAudioCategory(value1, value2 AudioCategory) (bool, error) {
	equal, err := AudioCategoryOperatorEqualityByAudioCategoryAndAudioCategory(value1, value2)
	if err != nil {
		return false, err
	}
	return !equal, nil
}

// disposeFromEngine releases the category's handle when its engine is disposed.
// It is unexported because AudioCategory has no Dispose in the contract and
// must not gain one.
func (c AudioCategory) disposeFromEngine() error {
	if c.resource == nil {
		return nil
	}
	return c.resource.Dispose()
}
