package main

import (
	"fmt"
	"sort"
	"strings"
)

// ---------------------------------------------------------------------------
// Foundation 74 — the measured frontier.
//
// This file answers "what is left, and why" as a MEASUREMENT rather than as
// prose. Every type the strict verifier reports missing must appear in exactly
// one family below, and every family must carry a classification. A missing
// type nobody has classified is UNREVIEWED, and UNREVIEWED is a defect: it is
// the difference between "we decided not to do this yet, for this reason" and
// "nobody looked".
//
// The registry exists because ROADMAP.md used to carry these counts by hand,
// and a hand-written count drifts the moment a milestone lands. ROADMAP.md's
// scoreboard and family table are now validated against this registry and
// against docs/generated/api-compat-report.json, and docs/generated/
// remaining-work.md is generated from it.
// ---------------------------------------------------------------------------

// Frontier classifications. The first is the only one that means "work this
// session could do"; every other one is a stop condition that must name what
// it is stopped on.
const (
	// frontierActionableLocal means nothing external blocks the family: the
	// reference is readable, the dependencies are projected, and the work is
	// implementation.
	frontierActionableLocal = "ACTIONABLE_LOCAL"
	// frontierBlockedUpstreamCNA means the native contract does not expose
	// what the member needs, and the exact route or state is named.
	frontierBlockedUpstreamCNA = "BLOCKED_UPSTREAM_CNA"
	// frontierBlockedPlatform means the reference behaviour belongs to a
	// platform subsystem this profile's host does not have.
	frontierBlockedPlatform = "BLOCKED_PLATFORM"
	// frontierBlockedHardware means a physical device is required and none is
	// present, or opening one needs an authorization nobody has given.
	frontierBlockedHardware = "BLOCKED_HARDWARE"
	// frontierBlockedFixture means the behaviour needs an asset the project
	// cannot yet author legally or deterministically.
	frontierBlockedFixture = "BLOCKED_FIXTURE"
	// frontierBlockedReferenceAsset means the pinned reference material needed
	// to establish the behaviour is absent.
	frontierBlockedReferenceAsset = "BLOCKED_REFERENCE_ASSET"
	// frontierLanguageMappingLimitation means Go cannot express the operation,
	// measured member by member rather than claimed for a family.
	frontierLanguageMappingLimitation = "LANGUAGE_MAPPING_LIMITATION"
	// frontierBCLProjectionBlockedExternal means the type needs a BCL closure
	// the profile names nowhere else, and the closure is measured.
	frontierBCLProjectionBlockedExternal = "BCL_PROJECTION_BLOCKED_EXTERNAL"
	// frontierDeliberateNonBinding means the surface exists and CNA-Go
	// deliberately does not bind it, with the reason recorded.
	frontierDeliberateNonBinding = "DELIBERATE_NON_BINDING"
	// frontierUnreviewed is the absence of a decision. It is never written by
	// hand: the exhaustiveness check assigns it to a missing type no family
	// claims, so it can only ever be nonzero because something was forgotten.
	frontierUnreviewed = "UNREVIEWED"
)

// fullProfileReferenceTypes is the pinned type count of the XNA 4.0 Windows
// runtime contract. The frontier measurement runs only when the surface under
// test is that whole profile, because partitioning the missing-type set of a
// single-type fixture is not a claim about anything.
const fullProfileReferenceTypes = 257

var frontierClassifications = map[string]bool{
	frontierActionableLocal:              true,
	frontierBlockedUpstreamCNA:           true,
	frontierBlockedPlatform:              true,
	frontierBlockedHardware:              true,
	frontierBlockedFixture:               true,
	frontierBlockedReferenceAsset:        true,
	frontierLanguageMappingLimitation:    true,
	frontierBCLProjectionBlockedExternal: true,
	frontierDeliberateNonBinding:         true,
	frontierUnreviewed:                   true,
}

// frontierFamily is one group of still-missing types that share a reason.
type frontierFamily struct {
	// Name is the family's short label, used in ROADMAP.md's table.
	Name string
	// Classification is one of the constants above.
	Classification string
	// Blocker names what the classification is stopped on. It is required for
	// every classification except ACTIONABLE_LOCAL, where "nothing external"
	// is the whole claim.
	Blocker string
	// Note is the measured detail: what the family needs first, which native
	// routes exist, or which decision it waits on.
	Note string
	// Types is the exact reference identity of every member of the family.
	Types []string
}

// frontierFamilies partitions the profile's still-missing types.
//
// The list is ordered the way the work is: dependencies first. It is checked
// against the live missing-type set on every run, so a family that becomes
// empty, or a type that arrives without a family, is a verifier failure rather
// than a stale paragraph.
var frontierFamilies = []frontierFamily{
	{
		Name:           "XACT",
		Classification: frontierActionableLocal,
		Note:           "~32/12/10/17 CNA routes; needs project-authored legal bank fixtures, and structural state is not audibility",
		Types: []string{
			"Microsoft.Xna.Framework.Audio.AudioCategory",
			"Microsoft.Xna.Framework.Audio.AudioEngine",
			"Microsoft.Xna.Framework.Audio.Cue",
			"Microsoft.Xna.Framework.Audio.SoundBank",
			"Microsoft.Xna.Framework.Audio.WaveBank",
		},
	},
	{
		Name:           "Media and video",
		Classification: frontierActionableLocal,
		Note:           "~31/23/53/46/50/55/19 CNA routes; the family splits into pure collection semantics, library enumeration, playback and video decode, and an empty host library is valid evidence for empty collections rather than for unimplementable ones",
		Types: []string{
			"Microsoft.Xna.Framework.Media.Album",
			"Microsoft.Xna.Framework.Media.AlbumCollection",
			"Microsoft.Xna.Framework.Media.Artist",
			"Microsoft.Xna.Framework.Media.ArtistCollection",
			"Microsoft.Xna.Framework.Media.Genre",
			"Microsoft.Xna.Framework.Media.GenreCollection",
			"Microsoft.Xna.Framework.Media.MediaLibrary",
			"Microsoft.Xna.Framework.Media.MediaPlayer",
			"Microsoft.Xna.Framework.Media.MediaQueue",
			"Microsoft.Xna.Framework.Media.MediaSource",
			"Microsoft.Xna.Framework.Media.Picture",
			"Microsoft.Xna.Framework.Media.PictureAlbum",
			"Microsoft.Xna.Framework.Media.PictureAlbumCollection",
			"Microsoft.Xna.Framework.Media.PictureCollection",
			"Microsoft.Xna.Framework.Media.Playlist",
			"Microsoft.Xna.Framework.Media.PlaylistCollection",
			"Microsoft.Xna.Framework.Media.Song",
			"Microsoft.Xna.Framework.Media.SongCollection",
			"Microsoft.Xna.Framework.Media.Video",
			"Microsoft.Xna.Framework.Media.VideoPlayer",
		},
	},
	{
		Name:           "GamerServices",
		Classification: frontierActionableLocal,
		Note:           "one type over ~53 guide_* CNA routes; sign-in-dependent behaviour is a separate question from the component's Initialize/Update ordering",
		Types:          []string{"Microsoft.Xna.Framework.GamerServices.GamerServicesComponent"},
	},
	{
		Name:           "Content plumbing",
		Classification: frontierActionableLocal,
		Note:           "68 cna_content* routes in total -- 20 reader, 13 type-reader, 34 manager, of which only 3 are _ext -- and the question is how much of ContentReader's public state, stream position, shared resources, external references and type-reader identity, the C API actually exposes, measured member by member rather than blocked as a family. This family also unblocks the Model family's native slice: Model has no public constructor, so ContentManager.Load<Model> is the only way a consumer can obtain one to draw",
		Types: []string{
			"Microsoft.Xna.Framework.Content.ContentReader",
			"Microsoft.Xna.Framework.Content.ContentTypeReader",
			"Microsoft.Xna.Framework.Content.ContentTypeReader`1",
			"Microsoft.Xna.Framework.Content.ContentTypeReaderManager",
			"Microsoft.Xna.Framework.Content.ResourceContentManager",
		},
	},
	{
		Name:           "Content serializer attributes",
		Classification: frontierActionableLocal,
		Note:           "the five attribute TYPES are ordinary classes with constructors and string/bool properties; whether Go can ATTACH them to declarations is a different question from whether the types can exist, and only the attaching operation is a candidate language limitation",
		Types: []string{
			"Microsoft.Xna.Framework.Content.ContentSerializerAttribute",
			"Microsoft.Xna.Framework.Content.ContentSerializerCollectionItemNameAttribute",
			"Microsoft.Xna.Framework.Content.ContentSerializerIgnoreAttribute",
			"Microsoft.Xna.Framework.Content.ContentSerializerRuntimeTypeAttribute",
			"Microsoft.Xna.Framework.Content.ContentSerializerTypeVersionAttribute",
		},
	},
	{
		Name:           "Design converters",
		Classification: frontierActionableLocal,
		Note:           "thirteen pure-managed types with no native dependency at all; they need the minimal System.ComponentModel closure their IL actually reaches -- TypeConverter, ExpandableObjectConverter, ITypeDescriptorContext, PropertyDescriptor(Collection), InstanceDescriptor and CultureInfo -- and nothing more",
		Types: []string{
			"Microsoft.Xna.Framework.Design.BoundingBoxConverter",
			"Microsoft.Xna.Framework.Design.BoundingSphereConverter",
			"Microsoft.Xna.Framework.Design.ColorConverter",
			"Microsoft.Xna.Framework.Design.MathTypeConverter",
			"Microsoft.Xna.Framework.Design.MatrixConverter",
			"Microsoft.Xna.Framework.Design.PlaneConverter",
			"Microsoft.Xna.Framework.Design.PointConverter",
			"Microsoft.Xna.Framework.Design.QuaternionConverter",
			"Microsoft.Xna.Framework.Design.RayConverter",
			"Microsoft.Xna.Framework.Design.RectangleConverter",
			"Microsoft.Xna.Framework.Design.Vector2Converter",
			"Microsoft.Xna.Framework.Design.Vector3Converter",
			"Microsoft.Xna.Framework.Design.Vector4Converter",
		},
	},
}

// frontierMeasurement is one family's live row.
type frontierMeasurement struct {
	Family         string   `json:"family"`
	Classification string   `json:"classification"`
	Blocker        string   `json:"blocker,omitempty"`
	Note           string   `json:"note"`
	Types          []string `json:"types"`
	// Remaining is how many of the family's types the live run still reports
	// missing. A family whose Remaining is zero has been closed and its entry
	// must be deleted rather than left as a monument.
	Remaining int    `json:"remaining"`
	Verdict   string `json:"verdict"`
}

// measureFrontier partitions the live missing-type set across the registry and
// reports what each family is stopped on.
//
// Three claims are enforced, and each of them is a way the registry could
// silently rot:
//
//   - every missing type is claimed by exactly one family, so a type that
//     arrives without a decision is UNREVIEWED rather than invisible;
//   - no family claims a type that is not missing, so a family that has been
//     closed cannot keep asserting work that no longer exists;
//   - every non-ACTIONABLE_LOCAL family names its blocker, which is the same
//     rule Foundation 29 imposed on deferred BCL bases.
func measureFrontier(result *report, missingTypes []string) []frontierMeasurement {
	missing := make(map[string]bool, len(missingTypes))
	for _, name := range missingTypes {
		missing[name] = true
	}
	claimed := make(map[string]string, len(missing))
	measurements := make([]frontierMeasurement, 0, len(frontierFamilies))
	for _, family := range frontierFamilies {
		measurement := frontierMeasurement{
			Family: family.Name, Classification: family.Classification,
			Blocker: family.Blocker, Note: family.Note, Types: family.Types, Verdict: "PASS",
		}
		if !frontierClassifications[family.Classification] {
			addDiagnostic(result, diagnostic{
				Category: "UNMEASURED_STRUCTURAL_CATEGORY", XNA: family.Name,
				Message: fmt.Sprintf("frontier family declares the unknown classification %q", family.Classification),
			})
			measurement.Verdict = "FAIL"
		}
		if family.Classification != frontierActionableLocal && strings.TrimSpace(family.Blocker) == "" {
			addDiagnostic(result, diagnostic{
				Category: "UNMEASURED_STRUCTURAL_CATEGORY", XNA: family.Name,
				Message: fmt.Sprintf("frontier family is %s and names no blocker", family.Classification),
			})
			measurement.Verdict = "FAIL"
		}
		if strings.TrimSpace(family.Note) == "" {
			addDiagnostic(result, diagnostic{
				Category: "UNMEASURED_STRUCTURAL_CATEGORY", XNA: family.Name,
				Message: "frontier family records no measured note",
			})
			measurement.Verdict = "FAIL"
		}
		for _, name := range family.Types {
			if owner, already := claimed[name]; already {
				addDiagnostic(result, diagnostic{
					Category: "UNMEASURED_STRUCTURAL_CATEGORY", XNA: name,
					Message: fmt.Sprintf("type is claimed by both the %q and %q frontier families", owner, family.Name),
				})
				measurement.Verdict = "FAIL"
				continue
			}
			claimed[name] = family.Name
			if !missing[name] {
				addDiagnostic(result, diagnostic{
					Category: "UNMEASURED_STRUCTURAL_CATEGORY", XNA: name,
					Message: fmt.Sprintf("frontier family %q claims a type the verifier does not report missing", family.Name),
				})
				measurement.Verdict = "FAIL"
				continue
			}
			measurement.Remaining++
		}
		if measurement.Remaining == 0 {
			addDiagnostic(result, diagnostic{
				Category: "UNMEASURED_STRUCTURAL_CATEGORY", XNA: family.Name,
				Message: "frontier family has no remaining missing type and must be deleted rather than kept",
			})
			measurement.Verdict = "FAIL"
		}
		result.Summary["GLOBAL_"+family.Classification] += measurement.Remaining
		measurements = append(measurements, measurement)
	}
	var unreviewed []string
	for _, name := range missingTypes {
		if claimed[name] == "" {
			unreviewed = append(unreviewed, name)
		}
	}
	sort.Strings(unreviewed)
	result.Summary["GLOBAL_UNREVIEWED"] = len(unreviewed)
	for _, name := range unreviewed {
		addDiagnostic(result, diagnostic{
			Category: "UNMEASURED_STRUCTURAL_CATEGORY", XNA: name,
			Message: "missing type belongs to no frontier family, so nothing has decided what blocks it",
		})
	}
	if len(unreviewed) > 0 {
		measurements = append(measurements, frontierMeasurement{
			Family: "UNREVIEWED", Classification: frontierUnreviewed,
			Blocker: "nothing has classified these types",
			Note:    "assigned by the exhaustiveness check, never by hand",
			Types:   unreviewed, Remaining: len(unreviewed), Verdict: "FAIL",
		})
	}
	return measurements
}

// renderRemainingWork is the generated answer to "what is left". It replaces
// the hand-maintained family table ROADMAP.md used to carry.
func renderRemainingWork(measurements []frontierMeasurement, summary map[string]int) string {
	var out strings.Builder
	out.WriteString("# Generated remaining-work table\n\n")
	out.WriteString("Generated by `go run ./tools/api_compat`. Do not edit by hand.\n\n")
	out.WriteString(fmt.Sprintf("- Missing types: %d\n", summary["MISSING_TYPE"]))
	out.WriteString(fmt.Sprintf("- Missing members: %d\n", summary["MISSING_MEMBER"]))
	out.WriteString(fmt.Sprintf("- Partial types: %d\n", summary["PARTIAL_TYPES"]))
	out.WriteString(fmt.Sprintf("- GLOBAL_ACTIONABLE_LOCAL: %d\n", summary["GLOBAL_"+frontierActionableLocal]))
	out.WriteString(fmt.Sprintf("- GLOBAL_UNREVIEWED: %d\n\n", summary["GLOBAL_UNREVIEWED"]))
	out.WriteString("| family | types | classification | blocker |\n| --- | ---: | --- | --- |\n")
	for _, measurement := range measurements {
		blocker := measurement.Blocker
		if blocker == "" {
			blocker = "nothing external"
		}
		out.WriteString(fmt.Sprintf("| %s | %d | `%s` | %s |\n",
			measurement.Family, measurement.Remaining, measurement.Classification, blocker))
	}
	out.WriteString("\n## What each family is waiting on\n")
	for _, measurement := range measurements {
		out.WriteString(fmt.Sprintf("\n### %s — `%s`\n\n%s\n\n",
			measurement.Family, measurement.Classification, measurement.Note))
		for _, name := range measurement.Types {
			out.WriteString(fmt.Sprintf("- `%s`\n", name))
		}
	}
	return out.String()
}
