package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
)

// ---------------------------------------------------------------------------
// Milestone 55 — CLR object identity under composition.
// ---------------------------------------------------------------------------

// # The problem private named composition creates, and this measures
//
// In CLR, `ldarg.0` inside a base class's body is the WHOLE object. Composition
// splits that object in two, and the base half has no way back to the whole
// one. Every reference site that uses `ldarg.0` as an OBJECT -- an event
// sender, a collection identity, a ToString subject -- then means the base
// half, which is not the object the consumer holds:
//
//	GameComponent::Dispose(bool)
//	  Game.Components.Remove(this)     // removes the base; the collection
//	                                   // holds the DERIVED object
//	  Disposed(this, EventArgs.Empty)  // announces the base as the sender of
//	                                   // the derived object's event
//
// Both were live before Milestone 55 measured them: a DrawableGameComponent
// survived Game.Dispose still in Game.Components, and its three inherited
// events announced a sender no consumer could match.
//
// A site that uses `ldarg.0` merely to REACH A FIELD is not affected and is not
// listed: the state it reads is the base's own.
//
// # The projection
//
// The composed base holds an unexported reference to the outermost derived
// object, installed by the derived constructor and read only through an
// unexported accessor. It is not exported, not a public Base/Parent accessor,
// and not reachable from outside the module -- the settled composition rule is
// unchanged; this is what makes it CORRECT rather than merely private.
//
// # What is checked
//
// Structurally, from the Go syntax tree rather than from text:
//
//  1. every COMPOSED XNA base has an identity entry, so a future composed base
//     cannot skip the decision by saying nothing;
//  2. the Go base type declares the unexported derived-reference field;
//  3. the base declares the unexported bind and self members, both unexported;
//  4. every recorded identity site on the base REACHES the self accessor in its
//     own body -- a site that used the bare receiver would compile and would be
//     wrong, so this is the only mechanical way to hold it;
//  5. every projected derived type's constructor installs itself through the
//     bind member.
//
// Recording the sites is half the evidence: a site list that omits a real
// `ldarg.0` object use is a gap, so each entry carries the IL it comes from.
type xnaCompositionIdentity struct {
	// Package is the Go package path the base and its derived types live in.
	Package string
	// GoBase is the Go type carrying the base's state.
	GoBase string
	// DerivedField is the unexported field on GoBase holding the CLR `this`.
	DerivedField string
	// SelfMember is the unexported accessor that yields the CLR `this`.
	SelfMember string
	// BindMember is the unexported installer a derived constructor calls.
	BindMember string
	// ForwardsTo names the Go base this one hands its binding to, for a base
	// that is ITSELF composed over another. A three-deep chain has ONE CLR
	// `this` and one place that answers with it; a middle link that kept a copy
	// would be a second answer that could disagree. Texture is such a link: it
	// takes a bind and passes it to GraphicsResource, and holds no derived
	// field or self accessor of its own.
	//
	// Foundation 79 separated the two halves of that claim. A middle link must
	// hold no derived field and must declare no self accessor -- one holder per
	// chain -- but it may still have SITES, because a middle link is a class
	// with members of its own and one of those members can use the object.
	// Effect is the first: set_CurrentTechnique reports the runtime type in an
	// ObjectDisposedException, and on a BasicEffect that type is BasicEffect.
	// Such a link names in SelfMember the accessor it reaches THROUGH the link
	// it forwards to, and the verifier checks that it does not declare one.
	ForwardsTo string
	// Sites are the Go members of GoBase whose reference IL pushes `ldarg.0`
	// as an OBJECT. Every one must reach SelfMember.
	Sites []xnaCompositionIdentitySite
	// DerivedConstructors are the Go constructors that must install the CLR
	// `this`, keyed by the derived CLR identity. A derived type CNA-Go does not
	// project yet is absent, and its row reports NOT_PROJECTED.
	DerivedConstructors map[string]string
}

type xnaCompositionIdentitySite struct {
	// GoMember is the method on the Go base.
	GoMember string
	// Uses is how many times the reference body pushes an OBJECT whose runtime
	// identity or TYPE the member then uses. That is `ldarg.0` in every site
	// but one: a clone constructor checks its SOURCE with `ldarg.1`, and the
	// projection turns that argument into the receiver, so the same resolution
	// is needed and the same count applies. Each site's Reference names which.
	//
	// The Go body must reach the self accessor exactly that many times: a
	// member with two identity uses that reaches self once has one site still
	// spelled with the base half, which is a defect a "reaches it at all" test
	// cannot see.
	Uses int
	// Reference is the exact IL the site reproduces.
	Reference string
}

// xnaCompositionIdentities is the closed registry. Every COMPOSED XNA base
// relationship must appear here.
var xnaCompositionIdentities = map[string]xnaCompositionIdentity{
	"Microsoft.Xna.Framework.GameComponent": {
		Package:      modulePath + "/Microsoft/Xna/Framework",
		GoBase:       "GameComponent",
		DerivedField: "derived",
		SelfMember:   "self",
		BindMember:   "bindDerived",
		Sites: []xnaCompositionIdentitySite{
			{GoMember: "SetEnabled", Uses: 1, Reference: "set_Enabled: ldarg.0; ldarg.0; ldsfld EventArgs::Empty; callvirt OnEnabledChanged(object, EventArgs) -- the SECOND ldarg.0 is the sender argument"},
			{GoMember: "SetUpdateOrder", Uses: 1, Reference: "set_UpdateOrder: ldarg.0; ldarg.0; ldsfld EventArgs::Empty; callvirt OnUpdateOrderChanged(object, EventArgs)"},
			{GoMember: "OnEnabledChanged", Uses: 1, Reference: "OnEnabledChanged: ldfld EnabledChanged; ldarg.0; ldarg.2; callvirt EventHandler`1::Invoke(object, !0) -- the raise ignores its own sender parameter and pushes `this`"},
			{GoMember: "OnUpdateOrderChanged", Uses: 1, Reference: "OnUpdateOrderChanged: ldfld UpdateOrderChanged; ldarg.0; ldarg.2; callvirt EventHandler`1::Invoke(object, !0)"},
			{GoMember: "DisposeByBoolean", Uses: 2, Reference: "Dispose(bool): ldarg.0; callvirt Collection`1::Remove(!0) -- AND -- ldfld Disposed; ldarg.0; ldsfld EventArgs::Empty; callvirt Invoke(object, !0)"},
		},
		DerivedConstructors: map[string]string{
			"Microsoft.Xna.Framework.DrawableGameComponent": "NewDrawableGameComponent",
		},
	},

	// Foundation 56. GraphicsResource carries two identity sites, and they are
	// two DIFFERENT uses of `ldarg.0` as an object: one needs the object, the
	// other needs its TYPE.
	"Microsoft.Xna.Framework.Graphics.GraphicsResource": {
		Package:      modulePath + "/Microsoft/Xna/Framework/Graphics",
		GoBase:       "GraphicsResource",
		DerivedField: "derived",
		SelfMember:   "self",
		BindMember:   "bindDerived",
		Sites: []xnaCompositionIdentitySite{
			{GoMember: "ToString", Uses: 1, Reference: "ToString: the name is empty, so `call instance string System.Object::ToString()` on `ldarg.0` -- which answers with the RUNTIME type's full CLR name, so a Texture2D must not answer with GraphicsResource's"},
			{GoMember: "DisposeByBoolean", Uses: 1, Reference: "~GraphicsResource(): ldfld <backing_store>Disposing; ldarg.0; ldsfld EventArgs::Empty; callvirt EventHandler`1::Invoke(object, !0)"},
		},
		DerivedConstructors: map[string]string{
			"Microsoft.Xna.Framework.Graphics.Texture":     "newTexture",
			"Microsoft.Xna.Framework.Graphics.SpriteBatch": "NewSpriteBatch",
			// The four state objects. Each has ONE public constructor, and the
			// private preset constructor the static instances use forwards to
			// it, so binding once in the public one covers every instance.
			"Microsoft.Xna.Framework.Graphics.BlendState":        "NewBlendState",
			"Microsoft.Xna.Framework.Graphics.DepthStencilState": "NewDepthStencilState",
			"Microsoft.Xna.Framework.Graphics.RasterizerState":   "NewRasterizerState",
			"Microsoft.Xna.Framework.Graphics.SamplerState":      "NewSamplerState",
			// Foundation 64. VertexDeclaration has TWO public constructors and
			// the registry names the shared unexported one both delegate to,
			// which is the same shape the state objects' preset constructor
			// has: binding once where the object is actually built covers
			// every path that builds one.
			"Microsoft.Xna.Framework.Graphics.VertexDeclaration": "newVertexDeclaration",
			// Foundation 65. IndexBuffer's two public constructors both reach
			// the same unexported builder, because the Type-keyed one is
			// literally `call .ctor(device, sizeForType(type), count, usage)`
			// in the reference too.
			"Microsoft.Xna.Framework.Graphics.IndexBuffer": "newIndexBuffer",
			// Foundation 66. Both public constructors reach the same builder;
			// the Type-keyed one resolves a declaration first and then calls
			// the other, which is what the reference's does too.
			"Microsoft.Xna.Framework.Graphics.VertexBuffer": "newVertexBuffer",
			// Foundation 83. OcclusionQuery has one public constructor and no
			// other door: unlike Effect it is not content-loadable, and unlike
			// the state objects it has no preset instances.
			"Microsoft.Xna.Framework.Graphics.OcclusionQuery": "NewOcclusionQuery",
			// Foundation 72. Effect has TWO doors -- its public compiled-bytecode
			// constructor and ContentManager.Load<Effect> -- and both reach
			// newEffect, which is where the whole reflected graph is built and
			// where the CLR `this` is installed.
			"Microsoft.Xna.Framework.Graphics.Effect": "newEffect",
		},
	},

	// Texture is the middle link. It has no identity site of its own -- both of
	// its members are plain field reads -- and holds no copy of the CLR `this`:
	// it forwards.
	"Microsoft.Xna.Framework.Graphics.Texture": {
		Package:    modulePath + "/Microsoft/Xna/Framework/Graphics",
		GoBase:     "Texture",
		BindMember: "bindDerived",
		ForwardsTo: "GraphicsResource",
		DerivedConstructors: map[string]string{
			"Microsoft.Xna.Framework.Graphics.Texture2D": "newTexture2D",
			// Foundation 71. Both are unexported for the reason
			// newTexture2D is: the projection of InitializeDescription, which
			// the exported constructor calls once the native object exists.
			"Microsoft.Xna.Framework.Graphics.Texture3D":   "newTexture3D",
			"Microsoft.Xna.Framework.Graphics.TextureCube": "newTextureCube",
		},
	},

	// Foundation 73. TextureCube is the third middle link, and the chain it
	// heads is four deep too: RenderTargetCube -> TextureCube -> Texture ->
	// GraphicsResource. It has no identity site of its own -- its one member is
	// a field read -- and holds no copy of the CLR `this`: it forwards, exactly
	// as Texture and Texture2D do.
	"Microsoft.Xna.Framework.Graphics.TextureCube": {
		Package:    modulePath + "/Microsoft/Xna/Framework/Graphics",
		GoBase:     "TextureCube",
		BindMember: "bindDerived",
		ForwardsTo: "Texture",
		DerivedConstructors: map[string]string{
			"Microsoft.Xna.Framework.Graphics.RenderTargetCube": "NewRenderTargetCubeByGraphicsDeviceAndInt32AndBooleanAndSurfaceFormatAndDepthFormatAndInt32AndRenderTargetUsage",
		},
	},

	// Foundation 79. Effect is the first middle link with SITES of its own, and
	// the chain it heads is three deep: BasicEffect -> Effect ->
	// GraphicsResource.
	//
	// It holds no derived field and declares no self accessor -- it passes its
	// binding on -- but two of its members report a CLR TYPE, and on a
	// BasicEffect that type is BasicEffect. Both reach GraphicsResource's self
	// accessor through the composed base.
	"Microsoft.Xna.Framework.Graphics.Effect": {
		Package:    modulePath + "/Microsoft/Xna/Framework/Graphics",
		GoBase:     "Effect",
		SelfMember: "self",
		BindMember: "bindDerived",
		ForwardsTo: "GraphicsResource",
		Sites: []xnaCompositionIdentitySite{
			{GoMember: "SetCurrentTechnique", Uses: 1, Reference: "set_CurrentTechnique: ldarg.0; ldfld pComPtr; ... ldarg.0; call Helpers::CheckDisposed(object, native int) -- the object decides the ObjectDisposedException's type name"},
			{GoMember: "cloneBase", Uses: 1, Reference: ".ctor(Effect cloneSource): ldarg.1; ldfld pComPtr; ... ldarg.1; call Helpers::CheckDisposed(object, native int) -- the object is the clone SOURCE, which the projection's cloneBase takes as its receiver, so the same resolution applies through it"},
		},
		DerivedConstructors: map[string]string{
			"Microsoft.Xna.Framework.Graphics.BasicEffect": "NewBasicEffectByGraphicsDevice",
			// Foundation 80. Each names its DEVICE constructor; the clone
			// constructor installs the binding too, and binding once where the
			// object is built covers every path that builds one -- the same
			// shape VertexDeclaration's and IndexBuffer's entries have.
			"Microsoft.Xna.Framework.Graphics.AlphaTestEffect":   "NewAlphaTestEffectByGraphicsDevice",
			"Microsoft.Xna.Framework.Graphics.DualTextureEffect": "NewDualTextureEffectByGraphicsDevice",
			// EffectMaterial has ONE constructor and it takes a source effect,
			// because the reference's whole body is `base(cloneSource)`.
			"Microsoft.Xna.Framework.Graphics.EffectMaterial": "NewEffectMaterial",
			// Foundation 81, the last two.
			"Microsoft.Xna.Framework.Graphics.EnvironmentMapEffect": "NewEnvironmentMapEffectByGraphicsDevice",
			"Microsoft.Xna.Framework.Graphics.SkinnedEffect":        "NewSkinnedEffectByGraphicsDevice",
		},
	},

	// Texture2D is the second middle link, and the chain is now four deep:
	// RenderTarget2D -> Texture2D -> Texture -> GraphicsResource. Every link but
	// the root forwards, so a RenderTarget2D's ToString answers with ITS name
	// after three hops.
	"Microsoft.Xna.Framework.Graphics.Texture2D": {
		Package:    modulePath + "/Microsoft/Xna/Framework/Graphics",
		GoBase:     "Texture2D",
		BindMember: "bindDerived",
		ForwardsTo: "Texture",
		DerivedConstructors: map[string]string{
			"Microsoft.Xna.Framework.Graphics.RenderTarget2D": "NewRenderTarget2DByGraphicsDeviceAndInt32AndInt32AndBooleanAndSurfaceFormatAndDepthFormatAndInt32AndRenderTargetUsage",
		},
	},
}

// measureXNACompositionIdentity holds the five claims above. It parses the
// package sources directly because the claim is about METHOD BODIES, which the
// declaration-level surface deliberately does not carry.
func measureXNACompositionIdentity(result *report, expected *expectedSurface, actual *actualSurface) []xnaCompositionIdentityMeasurement {
	if len(expected.XNABaseDerivedByBase) == 0 {
		return nil
	}
	bases := make([]string, 0, len(xnaBaseRelationships))
	for base := range xnaBaseRelationships {
		if xnaBaseRelationships[base].Status == "COMPOSED" {
			bases = append(bases, base)
		}
	}
	sort.Strings(bases)

	measurements := make([]xnaCompositionIdentityMeasurement, 0, len(bases))
	for _, base := range bases {
		measurement := xnaCompositionIdentityMeasurement{CLRBase: base, Verdict: "PASS"}
		fail := func(message string) {
			addDiagnostic(result, diagnostic{Category: "BASE_MAPPING_MISMATCH", XNA: base, Message: message})
			measurement.Verdict = "FAIL"
		}
		identity, recorded := xnaCompositionIdentities[base]
		if !recorded {
			fail("a COMPOSED XNA base records no object-identity entry; composition splits the CLR `this` and every composed base must state what it does about it")
			measurements = append(measurements, measurement)
			continue
		}
		measurement.GoBase = identity.GoBase
		measurement.DerivedField = identity.DerivedField
		dir, known := actual.PackageDirs[identity.Package]
		if !known {
			// A synthetic fixture carries no package directories. The registry
			// is still checked against the contract above; the body-level
			// claims need real sources and are skipped rather than faked.
			measurement.Verdict = "NOT_MEASURED"
			measurements = append(measurements, measurement)
			continue
		}
		bodies, fields, parseErr := parseGoPackageBodies(dir)
		if parseErr != nil {
			fail(fmt.Sprintf("the composed base's package could not be parsed: %v", parseErr))
			measurements = append(measurements, measurement)
			continue
		}

		measurement.ForwardsTo = identity.ForwardsTo
		if ast.IsExported(identity.DerivedField) || ast.IsExported(identity.SelfMember) || ast.IsExported(identity.BindMember) {
			fail("the object-identity mechanism is exported; it is private implementation state and the contract declares no accessor for the base object")
		}

		if identity.ForwardsTo != "" {
			// A middle link. It holds nothing of its own and must not: one CLR
			// `this`, one place that answers with it.
			if identity.DerivedField != "" {
				fail(fmt.Sprintf("%s forwards its binding to %s and also holds a derived field; one chain has one holder of the CLR `this`",
					identity.GoBase, identity.ForwardsTo))
			}
			// A middle link with sites names the accessor it reaches through
			// the link it forwards to, and must not declare one itself.
			switch {
			case len(identity.Sites) == 0 && identity.SelfMember != "":
				fail(fmt.Sprintf("%s records no identity site and still names a self accessor; a link with nothing to resolve names nothing",
					identity.GoBase))
			case len(identity.Sites) > 0 && identity.SelfMember == "":
				fail(fmt.Sprintf("%s records identity sites and names no self accessor, so nothing says what those sites must reach",
					identity.GoBase))
			case identity.SelfMember != "":
				if _, declared := bodies[identity.GoBase+"."+identity.SelfMember]; declared {
					fail(fmt.Sprintf("%s forwards its binding to %s and declares its own %s; that is a second answer to the one question the chain has",
						identity.GoBase, identity.ForwardsTo, identity.SelfMember))
				}
			}
			body, present := bodies[identity.GoBase+"."+identity.BindMember]
			switch {
			case !present:
				fail(fmt.Sprintf("%s declares no %s member", identity.GoBase, identity.BindMember))
			case !callsMethod(body, identity.BindMember):
				fail(fmt.Sprintf("%s.%s does not pass the binding on to %s; a middle link that swallows it leaves the chain answering with the wrong object",
					identity.GoBase, identity.BindMember, identity.ForwardsTo))
			default:
				result.Summary["XNA_COMPOSED_IDENTITY_FORWARDS"]++
			}
		} else {
			// (2) the unexported derived-reference field.
			if !fields[identity.GoBase][identity.DerivedField] {
				fail(fmt.Sprintf("%s declares no unexported %s field; a composed base cannot reach the CLR `this` without one",
					identity.GoBase, identity.DerivedField))
			}

			// (3) the two unexported members, and the field each must touch. A
			// self accessor that never reads the derived reference, or a bind
			// that never writes it, is a mechanism that compiles and does
			// nothing.
			for _, member := range []string{identity.SelfMember, identity.BindMember} {
				body, present := bodies[identity.GoBase+"."+member]
				if !present {
					fail(fmt.Sprintf("%s declares no %s member", identity.GoBase, member))
					continue
				}
				if !selectsField(body, identity.DerivedField) {
					fail(fmt.Sprintf("%s.%s never touches %s; the object-identity mechanism would compile and hold nothing",
						identity.GoBase, member, identity.DerivedField))
				}
			}
			if len(identity.Sites) == 0 {
				fail(fmt.Sprintf("%s holds the CLR `this` and records no identity site; a base that needs none forwards instead of holding one",
					identity.GoBase))
			}
		}

		// (4) every identity site reaches the self accessor.
		for _, site := range identity.Sites {
			row := xnaCompositionIdentitySiteRow{GoMember: site.GoMember, Reference: site.Reference, Uses: site.Uses}
			body, present := bodies[identity.GoBase+"."+site.GoMember]
			switch {
			case site.Uses < 1:
				fail(fmt.Sprintf("the identity registry records %s.%s with no `ldarg.0` object use; a site with none is not a site",
					identity.GoBase, site.GoMember))
			case !present:
				fail(fmt.Sprintf("%s declares no %s, so the identity site it carries cannot be checked", identity.GoBase, site.GoMember))
			default:
				row.Reaches = countCalls(body, identity.SelfMember)
				if row.Reaches != site.Uses {
					fail(fmt.Sprintf("%s.%s reaches %s %d times and the reference pushes `ldarg.0` as an OBJECT %d times there (%s); the bare receiver is the BASE half of a composed object",
						identity.GoBase, site.GoMember, identity.SelfMember, row.Reaches, site.Uses, site.Reference))
					break
				}
				result.Summary["XNA_COMPOSED_IDENTITY_SITES"]++
				result.Summary["XNA_COMPOSED_IDENTITY_USES"] += site.Uses
			}
			measurement.Sites = append(measurement.Sites, row)
		}

		// (5) every projected derived type installs itself.
		for _, derivedName := range expected.XNABaseDerivedByBase[base] {
			derived := expected.typeForXNA(derivedName)
			if derived == nil {
				continue
			}
			if _, projected := actual.Types[derived.Key]; !projected {
				continue
			}
			row := xnaCompositionIdentityBindingRow{Derived: derivedName, GoDerived: derived.GoName}
			constructor, named := identity.DerivedConstructors[derivedName]
			if !named {
				fail(fmt.Sprintf("%s is projected and the identity registry names no constructor for it", derivedName))
				measurement.Bindings = append(measurement.Bindings, row)
				continue
			}
			row.Constructor = constructor
			body, present := bodies["."+constructor]
			switch {
			case !present:
				fail(fmt.Sprintf("the identity registry names %s as %s's constructor and the package declares no such function", constructor, derivedName))
			case callsMethod(body, identity.BindMember):
				row.Binds = true
				result.Summary["XNA_COMPOSED_IDENTITY_BINDINGS"]++
			default:
				fail(fmt.Sprintf("%s does not call %s; a derived object that never installs itself leaves the composed base announcing and removing the BASE half",
					constructor, identity.BindMember))
			}
			measurement.Bindings = append(measurement.Bindings, row)
		}
		measurements = append(measurements, measurement)
	}
	return measurements
}

// parseGoPackageBodies returns every function and method body in one package
// directory, keyed "Receiver.Name" (a bare ".Name" for a package function), and
// the unexported field names of every struct type it declares.
func parseGoPackageBodies(dir string) (map[string]*ast.BlockStmt, map[string]map[string]bool, error) {
	filenames, err := goFiles(dir)
	if err != nil {
		return nil, nil, err
	}
	bodies := make(map[string]*ast.BlockStmt)
	fields := make(map[string]map[string]bool)
	fset := token.NewFileSet()
	for _, filename := range filenames {
		file, parseErr := parser.ParseFile(fset, filename, nil, parser.SkipObjectResolution)
		if parseErr != nil {
			return nil, nil, parseErr
		}
		for _, decl := range file.Decls {
			switch typed := decl.(type) {
			case *ast.FuncDecl:
				if typed.Body == nil {
					continue
				}
				bodies[receiverName(receiverExpr(typed))+"."+typed.Name.Name] = typed.Body
			case *ast.GenDecl:
				for _, spec := range typed.Specs {
					typeSpec, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}
					structType, ok := typeSpec.Type.(*ast.StructType)
					if !ok {
						continue
					}
					names := make(map[string]bool)
					for _, field := range structType.Fields.List {
						for _, name := range field.Names {
							names[name.Name] = true
						}
					}
					fields[typeSpec.Name.Name] = names
				}
			}
		}
	}
	return bodies, fields, nil
}

func receiverExpr(decl *ast.FuncDecl) ast.Expr {
	if decl.Recv == nil || len(decl.Recv.List) == 0 {
		return nil
	}
	return decl.Recv.List[0].Type
}

// countCalls is how many calls in a body have name as their selector. It walks
// the syntax tree rather than searching text, so a mention in a comment or a
// string is not a call -- the same rule the native reachability gate settled on
// after its own regexp implementation was fooled by prose.
func countCalls(body *ast.BlockStmt, name string) int {
	count := 0
	ast.Inspect(body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		if selector, ok := call.Fun.(*ast.SelectorExpr); ok && selector.Sel.Name == name {
			count++
		}
		return true
	})
	return count
}

func callsMethod(body *ast.BlockStmt, name string) bool { return countCalls(body, name) > 0 }

// selectsField reports whether a body reads or writes a field of that name
// through a selector, which is what both halves of the mechanism must do.
func selectsField(body *ast.BlockStmt, name string) bool {
	found := false
	ast.Inspect(body, func(node ast.Node) bool {
		if found {
			return false
		}
		selector, ok := node.(*ast.SelectorExpr)
		if ok && selector.Sel.Name == name {
			found = true
			return false
		}
		return true
	})
	return found
}
