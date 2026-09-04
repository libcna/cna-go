package framework

// CultureInfo projects System.Globalization.CultureInfo, and only as far as the
// profile reaches it.
//
// # Reference authority
//
//	mscorlib.dll  4.0.30319.1 (RTMRel.030319-0100), assembly version 4.0.0.0
//	sha256 5634668d4775b0113f08ea31093b281fea69bfc4e99227f5ca761b4ed98acc63
//
// # What the profile actually uses, measured
//
// Across the whole Microsoft.Xna.Framework.Design namespace, CultureInfo is
// touched through exactly two members:
//
//	CultureInfo::get_CurrentCulture     4 call sites, all `culture == null`
//	CultureInfo::get_TextInfo           4 call sites, every one followed by
//	TextInfo::get_ListSeparator         get_ListSeparator and nothing else
//
// So a CultureInfo, in this profile, is one thing: WHAT SEPARATES LIST ITEMS.
// The converters split "1, 2, 3" on the culture's separator followed by a
// space, and join with the same. Nothing reads a calendar, a number format or a
// name.
//
// It is a projected type rather than a bare string because it occupies
// twenty-one PARAMETER positions a consumer passes and a nil means "the current
// culture" -- a distinction a string cannot carry. The precedent is
// System.IAsyncResult, projected as *AsyncResult for the same reason: a BCL
// type the profile names in signatures gets a Go type, sized to what the
// profile reaches.
type CultureInfo struct {
	// listSeparator is TextInfo::get_ListSeparator. It is the only piece of a
	// culture this projection carries, and holding it directly rather than a
	// TextInfo keeps a type nothing else in the profile names out of the
	// surface.
	listSeparator string
}

// invariantListSeparator is what the invariant culture -- and every culture the
// .NET Framework ships for an English locale -- answers.
const invariantListSeparator = ","

// currentCulture is the process-wide CultureInfo::get_CurrentCulture.
//
// The reference reads the THREAD's culture, which the CLR initialises from the
// operating system and a program may reassign. Go has no thread culture and
// this projection does not invent one: the value is fixed at the invariant
// separator, which is what a Go process's locale-free formatting already
// assumes.
//
// That is a divergence and it is recorded rather than hidden. It shows only for
// a program that would have set a culture whose list separator is not a comma
// -- and no member of this profile can set one, because CultureInfo has no
// projected constructor and none is declared in the contract.
var currentCulture = &CultureInfo{listSeparator: invariantListSeparator}

// CultureInfoCurrentCulture is CultureInfo::get_CurrentCulture, the static the
// converters fall back to for a nil culture.
//
// It is a package function rather than a method because the CLR member is
// static, which is the settled spelling for a static in this projection.
func CultureInfoCurrentCulture() *CultureInfo { return currentCulture }

// ListSeparator is what TextInfo::get_ListSeparator answers for this culture,
// reached in the reference through CultureInfo::get_TextInfo.
//
// TextInfo is NOT projected: it would be a type the profile names in no
// signature and uses for exactly one property, so the property is carried here
// and the intermediate object is not. That is the same judgement that mapped
// System.Resources.ResourceManager to the one member the profile calls.
func (c *CultureInfo) ListSeparator() string {
	if c == nil {
		return currentCulture.listSeparator
	}
	return c.listSeparator
}

// cultureOrCurrent is the `if (culture == null) culture = CultureInfo.CurrentCulture`
// every converter opens with. It is unexported because the reference performs
// it inline in each member rather than exposing a helper.
func cultureOrCurrent(culture *CultureInfo) *CultureInfo {
	if culture == nil {
		return currentCulture
	}
	return culture
}
