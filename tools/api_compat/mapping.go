package main

import (
	"fmt"
	"hash/fnv"
	"path"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

const modulePath = "github.com/openeggbert/cna-go"

var bclTypes = map[string]string{
	"System.Boolean":   "bool",
	"System.Byte":      "uint8",
	"System.SByte":     "int8",
	"System.Int16":     "int16",
	"System.UInt16":    "uint16",
	"System.Int32":     "int32",
	"System.UInt32":    "uint32",
	"System.Int64":     "int64",
	"System.UInt64":    "uint64",
	"System.Single":    "float32",
	"System.Double":    "float64",
	"System.Char":      "uint16",
	"System.String":    "string",
	"System.Object":    "any",
	"System.IntPtr":    "uintptr",
	"System.TimeSpan":  "TimeSpan",
	"System.Type":      "reflect.Type",
	"System.IO.Stream": "io.Reader",
}

var operatorNames = map[string]string{
	"op_Addition":      "Addition",
	"op_Subtraction":   "Subtraction",
	"op_Multiply":      "Multiply",
	"op_Division":      "Division",
	"op_UnaryNegation": "UnaryNegation",
	"op_Equality":      "Equality",
	"op_Inequality":    "Inequality",
	"op_Implicit":      "Implicit",
	"op_Explicit":      "Explicit",
}

var managedTypes = map[string]bool{
	"Microsoft.Xna.Framework.MathHelper":      true,
	"Microsoft.Xna.Framework.Vector2":         true,
	"Microsoft.Xna.Framework.Vector3":         true,
	"Microsoft.Xna.Framework.Vector4":         true,
	"Microsoft.Xna.Framework.Quaternion":      true,
	"Microsoft.Xna.Framework.Matrix":          true,
	"Microsoft.Xna.Framework.Color":           true,
	"Microsoft.Xna.Framework.Point":           true,
	"Microsoft.Xna.Framework.Rectangle":       true,
	"Microsoft.Xna.Framework.Ray":             true,
	"Microsoft.Xna.Framework.Plane":           true,
	"Microsoft.Xna.Framework.BoundingBox":     true,
	"Microsoft.Xna.Framework.BoundingSphere":  true,
	"Microsoft.Xna.Framework.BoundingFrustum": true,
	"Microsoft.Xna.Framework.GameTime":        true,
}

func buildExpected(c contract) (*expectedSurface, error) {
	s := &expectedSurface{
		Types:          make(map[symbolKey]*expectedType),
		Members:        make(map[symbolKey]*expectedMember),
		ReferenceTypes: len(c.Types),
	}
	byIdentity := make(map[string]*contractType, len(c.Types))
	nameCollisions := make(map[string]int)
	for i := range c.Types {
		t := &c.Types[i]
		byIdentity[t.Name] = t
		nameCollisions[namespaceOf(t.Name)+"|"+flattenedBaseName(t.Name)]++
		s.ReferenceMembers += len(t.Members)
	}

	for i := range c.Types {
		t := &c.Types[i]
		goName := mappedTypeName(*t, nameCollisions)
		pkg := packagePathForNamespace(namespaceOf(t.Name))
		key := symbolKey{Package: pkg, Name: goName}
		if _, exists := s.Types[key]; exists {
			return nil, fmt.Errorf("mapped type collision at %s", key.String())
		}
		genericNames := make([]string, len(t.GenericParameters))
		for j, gp := range t.GenericParameters {
			genericNames[j] = gp.Name
		}
		s.Types[key] = &expectedType{
			Key: key, XNA: t.Name, GoName: goName, PackagePath: pkg,
			Kind: t.Kind, Flags: t.Flags, BaseType: valueOrEmpty(t.BaseType),
			Interfaces:       append([]string(nil), t.DirectInterfaces...),
			GenericParameter: genericNames,
		}
	}

	var allMembers []*expectedMember
	for i := range c.Types {
		t := &c.Types[i]
		owner := s.typeForXNA(t.Name)
		groups := overloadGroups(*t)
		for j := range t.Members {
			m := &t.Members[j]
			mapped := mapMember(s, byIdentity, owner, *t, *m, groups)
			allMembers = append(allMembers, mapped...)
		}
	}
	resolveMemberCollisions(allMembers)
	for _, em := range allMembers {
		if _, exists := s.Members[em.Key]; exists {
			return nil, fmt.Errorf("unresolved mapped member collision at %s from %s", em.Key.String(), em.XNA)
		}
		s.Members[em.Key] = em
		owner := s.typeForXNA(em.Owner)
		owner.Members = append(owner.Members, em.Key)
	}
	s.ExpectedGoTypes = len(s.Types)
	s.ExpectedGoMembers = len(s.Members)
	return s, nil
}

func resolveMemberCollisions(members []*expectedMember) {
	groups := make(map[symbolKey][]*expectedMember)
	for _, member := range members {
		groups[member.Key] = append(groups[member.Key], member)
	}
	for _, group := range groups {
		if len(group) < 2 {
			continue
		}
		for _, member := range group {
			member.GoName += collisionKindSuffix(member)
			member.Key.Name = member.GoName
		}
	}
	groups = make(map[symbolKey][]*expectedMember)
	for _, member := range members {
		groups[member.Key] = append(groups[member.Key], member)
	}
	for _, group := range groups {
		if len(group) < 2 {
			continue
		}
		for _, member := range group {
			member.GoName += "Signature" + signatureDigest(member.XNA)
			member.Key.Name = member.GoName
		}
	}
}

func collisionKindSuffix(member *expectedMember) string {
	if strings.Contains(member.XNA, "::op_") {
		return "Operator"
	}
	switch member.SourceKind {
	case "constructor":
		return "Constructor"
	case "field":
		return "Field"
	case "property":
		return "Property"
	case "event":
		return "Event"
	default:
		return "Method"
	}
}

func signatureDigest(identity string) string {
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(identity))
	return fmt.Sprintf("%08X", hash.Sum32())
}

func (s *expectedSurface) typeForXNA(identity string) *expectedType {
	for _, t := range s.Types {
		if t.XNA == identity {
			return t
		}
	}
	return nil
}

func overloadGroups(t contractType) map[string]int {
	result := make(map[string]int)
	constructors := 0
	for _, m := range t.Members {
		if m.Kind == "constructor" {
			constructors++
			continue
		}
		if m.Kind == "method" {
			result[fmt.Sprintf("%t|%s", m.Static, m.Name)]++
		}
	}
	result["constructors"] = constructors
	return result
}

func mapMember(s *expectedSurface, byIdentity map[string]*contractType, owner *expectedType, t contractType, m contractMember, groups map[string]int) []*expectedMember {
	xna := memberIdentity(t.Name, m)
	base := &expectedMember{XNA: xna, Owner: t.Name, SourceKind: m.Kind, PackagePath: owner.PackagePath, Receiver: owner.GoName}
	parameters, outResults, hasDirection := mapParameters(s, byIdentity, owner, m.Parameters)
	base.Parameters = parameters
	base.Results = mapReturn(s, byIdentity, owner, m.ReturnType)
	base.Results = append(base.Results, outResults...)
	if isFallible(t, m) {
		base.Results = append(base.Results, "error")
		base.ErrorAdded = true
	}
	shape := parameterShape(m.Parameters)

	switch m.Kind {
	case "constructor":
		name := "New" + owner.GoName
		if groups["constructors"] > 1 {
			name += "By" + shape
			base.OverloadMapped = true
		}
		base.GoName, base.GoKind, base.Receiver = name, "func", ""
		result := owner.GoName
		if t.Kind == "class" && t.Name != "Microsoft.Xna.Framework.GameTime" {
			result = "*" + result
		}
		base.Results = append([]string{result}, base.Results...)
		if t.Name == "Microsoft.Xna.Framework.Game" {
			base.Parameters = []string{"GameCallbacks"}
		}
	case "field":
		if t.Kind == "enum" && m.Name == "value__" {
			return nil
		}
		mappedType := mapType(s, byIdentity, owner, valueOrEmpty(m.Type))
		if t.Kind == "enum" {
			base.GoName, base.GoKind, base.Receiver = owner.GoName+m.Name, "const", ""
			base.Results = []string{owner.GoName}
			if len(m.Value) != 0 && string(m.Value) != "null" {
				v := strings.Trim(string(m.Value), "\"")
				base.EnumValue = &v
			}
		} else if m.Static {
			base.GoName, base.Receiver = owner.GoName+m.Name, ""
			if m.Constant {
				base.GoKind = "const"
				base.Results = []string{mappedType}
			} else {
				base.GoKind = "func"
				base.Parameters = nil
				base.Results = []string{mappedType}
			}
		} else {
			base.GoName, base.GoKind = m.Name, "field"
			base.Parameters = nil
			base.Results = []string{mappedType}
		}
	case "property":
		mappedType := mapType(s, byIdentity, owner, valueOrEmpty(m.Type))
		var result []*expectedMember
		if m.Get {
			get := cloneMember(base)
			get.GoKind = chooseMemberKind(m.Static)
			get.GoName = m.Name
			if m.Static {
				get.GoName = owner.GoName + m.Name
				get.Receiver = ""
			}
			get.Parameters = mapIndexerParameters(s, byIdentity, owner, m.Parameters)
			get.Results = mapResultType(s, byIdentity, owner, valueOrEmpty(m.Type))
			if isFallible(t, m) {
				get.Results = append(get.Results, "error")
				get.ErrorAdded = true
			}
			result = append(result, get)
		}
		if m.Set {
			set := cloneMember(base)
			set.GoKind = chooseMemberKind(m.Static)
			set.GoName = "Set" + m.Name
			if m.Static {
				set.GoName = "Set" + owner.GoName + m.Name
				set.Receiver = ""
			}
			set.Parameters = append(mapIndexerParameters(s, byIdentity, owner, m.Parameters), mappedType)
			set.Results = nil
			if isFallible(t, m) {
				set.Results = []string{"error"}
				set.ErrorAdded = true
			}
			result = append(result, set)
		}
		if target := descendantPropertyType(s, m); target != nil && target.PackagePath != owner.PackagePath && strings.HasPrefix(target.PackagePath, owner.PackagePath+"/") {
			for _, item := range result {
				item.PackagePath = target.PackagePath
				item.Receiver = ""
				item.GoKind = "func"
				if strings.HasPrefix(item.GoName, "Set") {
					item.GoName = "Set" + owner.GoName + m.Name
					item.Parameters = []string{"*framework." + owner.GoName, mapType(s, byIdentity, target, valueOrEmpty(m.Type))}
				} else {
					item.GoName = owner.GoName + m.Name
					item.Parameters = []string{"*framework." + owner.GoName}
					item.Results = []string{mapType(s, byIdentity, target, valueOrEmpty(m.Type))}
					if isFallible(t, m) {
						item.Results = append(item.Results, "error")
					}
				}
			}
		}
		for _, item := range result {
			item.Key = symbolKey{Package: item.PackagePath, Receiver: item.Receiver, Name: item.GoName}
		}
		return result
	case "method":
		if op, ok := operatorNames[m.Name]; ok {
			base.GoName = owner.GoName + "Operator" + op + "By" + shape
			base.GoKind, base.Receiver, base.OverloadMapped = "func", "", true
		} else {
			base.GoName = m.Name
			base.GoKind = chooseMemberKind(m.Static)
			if m.Static {
				base.GoName = owner.GoName + m.Name
				base.Receiver = ""
			}
			if groups[fmt.Sprintf("%t|%s", m.Static, m.Name)] > 1 {
				base.GoName += "By" + shape
				base.OverloadMapped = true
			}
		}
		if hasDirection {
			base.OverloadMapped = base.OverloadMapped || groups[fmt.Sprintf("%t|%s", m.Static, m.Name)] > 1
		}
	case "event":
		handler := mapType(s, byIdentity, owner, valueOrEmpty(m.Type))
		add := cloneMember(base)
		remove := cloneMember(base)
		add.GoName, remove.GoName = "Add"+m.Name+"Handler", "Remove"+m.Name+"Handler"
		add.GoKind, remove.GoKind = chooseMemberKind(m.Static), chooseMemberKind(m.Static)
		if m.Static {
			add.GoName, remove.GoName = owner.GoName+add.GoName, owner.GoName+remove.GoName
			add.Receiver, remove.Receiver = "", ""
		}
		add.Parameters, add.Results = []string{handler}, []string{"EventSubscription", "error"}
		remove.Parameters, remove.Results = []string{"EventSubscription"}, []string{"error"}
		add.Key = symbolKey{Package: add.PackagePath, Receiver: add.Receiver, Name: add.GoName}
		remove.Key = symbolKey{Package: remove.PackagePath, Receiver: remove.Receiver, Name: remove.GoName}
		return []*expectedMember{add, remove}
	default:
		return nil
	}
	if t.Name == "Microsoft.Xna.Framework.Game" && m.Kind == "method" && isGameLifecycleOverride(m.Name) {
		base.Receiver = "GameCallbacks"
		base.GoKind = "method"
		base.Parameters = append([]string{"*Game"}, base.Parameters...)
	}

	base.Key = symbolKey{Package: base.PackagePath, Receiver: base.Receiver, Name: base.GoName}
	return []*expectedMember{base}
}

func descendantPropertyType(surface *expectedSurface, member contractMember) *expectedType {
	if member.Type == nil {
		return nil
	}
	identity := strings.TrimSuffix(*member.Type, "&")
	if bracket := strings.Index(identity, "["); bracket >= 0 {
		identity = identity[:bracket]
	}
	return surface.typeForXNA(identity)
}

func isGameLifecycleOverride(name string) bool {
	switch name {
	case "Initialize", "LoadContent", "Update", "Draw", "UnloadContent":
		return true
	default:
		return false
	}
}

func cloneMember(in *expectedMember) *expectedMember {
	out := *in
	out.Parameters = append([]string(nil), in.Parameters...)
	out.Results = append([]string(nil), in.Results...)
	return &out
}

func chooseMemberKind(static bool) string {
	if static {
		return "func"
	}
	return "method"
}

func mapParameters(s *expectedSurface, byIdentity map[string]*contractType, owner *expectedType, params []contractParameter) ([]string, []string, bool) {
	var inputs, outputs []string
	hasDirection := false
	for _, p := range params {
		mapped := mapType(s, byIdentity, owner, p.Type)
		if p.Out {
			outputs = append(outputs, mapResultType(s, byIdentity, owner, p.Type)...)
			hasDirection = true
			continue
		}
		if p.Ref {
			mapped = "*" + strings.TrimPrefix(mapped, "*")
			hasDirection = true
		}
		inputs = append(inputs, mapped)
	}
	return inputs, outputs, hasDirection
}

func mapIndexerParameters(s *expectedSurface, byIdentity map[string]*contractType, owner *expectedType, params []contractParameter) []string {
	inputs, _, _ := mapParameters(s, byIdentity, owner, params)
	return inputs
}

func mapReturn(s *expectedSurface, byIdentity map[string]*contractType, owner *expectedType, raw *string) []string {
	if raw == nil || *raw == "System.Void" {
		return nil
	}
	return mapResultType(s, byIdentity, owner, *raw)
}

func mapResultType(s *expectedSurface, byIdentity map[string]*contractType, owner *expectedType, raw string) []string {
	if inner, ok := nullableInner(raw); ok {
		return []string{strings.TrimPrefix(mapType(s, byIdentity, owner, inner), "*"), "bool"}
	}
	return []string{mapType(s, byIdentity, owner, raw)}
}

func mapType(s *expectedSurface, byIdentity map[string]*contractType, owner *expectedType, raw string) string {
	raw = strings.TrimSuffix(raw, "&")
	if inner, ok := nullableInner(raw); ok {
		return "*" + strings.TrimPrefix(mapType(s, byIdentity, owner, inner), "*")
	}
	if strings.HasSuffix(raw, "[]") {
		return "[]" + mapType(s, byIdentity, owner, strings.TrimSuffix(raw, "[]"))
	}
	if mapped, ok := bclTypes[raw]; ok {
		return mapped
	}
	if raw == "System.Void" || raw == "" {
		return ""
	}
	if !strings.Contains(raw, ".") && !strings.Contains(raw, "[") {
		return raw
	}
	identity := raw
	if bracket := strings.Index(identity, "["); bracket >= 0 {
		identity = identity[:bracket]
	}
	identity = strings.TrimSuffix(identity, "&")
	ct, ok := byIdentity[identity]
	if !ok {
		if strings.HasPrefix(raw, "System.Collections.Generic.IEnumerable`1") || strings.HasPrefix(raw, "System.Collections.Generic.IList`1") || strings.HasPrefix(raw, "System.Collections.Generic.ICollection`1") {
			return "any"
		}
		return "any"
	}
	et := s.typeForXNA(ct.Name)
	name := et.GoName
	if et.PackagePath != owner.PackagePath {
		name = strings.ToLower(path.Base(et.PackagePath)) + "." + name
	}
	if ct.Kind == "class" && ct.Name != "Microsoft.Xna.Framework.GameTime" {
		name = "*" + name
	}
	return name
}

func nullableInner(raw string) (string, bool) {
	raw = strings.TrimSuffix(raw, "&")
	if !strings.HasPrefix(raw, "System.Nullable`1[") || !strings.HasSuffix(raw, "]") {
		return "", false
	}
	return strings.TrimSuffix(strings.TrimPrefix(raw, "System.Nullable`1["), "]"), true
}

func isFallible(t contractType, m contractMember) bool {
	if managedTypes[t.Name] || t.Kind == "enum" {
		return false
	}
	if m.Kind == "field" || m.Name == "ToString" || m.Name == "GetHashCode" || strings.HasPrefix(m.Name, "op_") {
		return false
	}
	return t.Kind == "class" || t.Kind == "interface"
}

func parameterShape(params []contractParameter) string {
	if len(params) == 0 {
		return "None"
	}
	parts := make([]string, 0, len(params))
	for _, p := range params {
		prefix := ""
		if p.Out {
			prefix = "Out"
		} else if p.Ref {
			prefix = "Ref"
		} else if p.In {
			prefix = "In"
		}
		parts = append(parts, prefix+typeShape(p.Type))
	}
	return strings.Join(parts, "And")
}

var nonIdentifier = regexp.MustCompile(`[^A-Za-z0-9]+`)

func typeShape(raw string) string {
	raw = strings.TrimSuffix(raw, "&")
	if strings.HasPrefix(raw, "System.Nullable`1[") && strings.HasSuffix(raw, "]") {
		inner := strings.TrimSuffix(strings.TrimPrefix(raw, "System.Nullable`1["), "]")
		return "NullableOf" + typeShape(inner)
	}
	array := strings.HasSuffix(raw, "[]")
	raw = strings.TrimSuffix(raw, "[]")
	if dot := strings.LastIndex(raw, "."); dot >= 0 {
		raw = raw[dot+1:]
	}
	raw = strings.ReplaceAll(raw, "+", "")
	raw = strings.ReplaceAll(raw, "`", "Of")
	raw = nonIdentifier.ReplaceAllString(raw, "")
	if array {
		raw = "SliceOf" + raw
	}
	return exportIdentifier(raw)
}

func memberIdentity(owner string, m contractMember) string {
	params := make([]string, len(m.Parameters))
	for i, p := range m.Parameters {
		direction := ""
		if p.Out {
			direction = "out "
		} else if p.Ref {
			direction = "ref "
		} else if p.In {
			direction = "in "
		}
		params[i] = direction + p.Type
	}
	return fmt.Sprintf("%s::%s(%s)", owner, m.Name, strings.Join(params, ","))
}

func namespaceOf(identity string) string {
	top := identity
	if plus := strings.Index(top, "+"); plus >= 0 {
		top = top[:plus]
	}
	dot := strings.LastIndex(top, ".")
	if dot < 0 {
		return ""
	}
	return top[:dot]
}

func flattenedBaseName(identity string) string {
	name := identity[strings.LastIndex(identity, ".")+1:]
	parts := strings.Split(name, "+")
	for i := range parts {
		if tick := strings.Index(parts[i], "`"); tick >= 0 {
			parts[i] = parts[i][:tick]
		}
	}
	return strings.Join(parts, "")
}

func mappedTypeName(t contractType, collisions map[string]int) string {
	base := flattenedBaseName(t.Name)
	if len(t.GenericParameters) > 0 && collisions[namespaceOf(t.Name)+"|"+base] > 1 {
		base += "Of"
		for _, p := range t.GenericParameters {
			base += exportIdentifier(p.Name)
		}
	}
	return exportIdentifier(base)
}

func exportIdentifier(value string) string {
	if value == "" {
		return value
	}
	runes := []rune(value)
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

func packagePathForNamespace(namespace string) string {
	return modulePath + "/" + strings.ReplaceAll(namespace, ".", "/")
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func sortedExpectedTypes(s *expectedSurface) []*expectedType {
	result := make([]*expectedType, 0, len(s.Types))
	for _, t := range s.Types {
		result = append(result, t)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].XNA < result[j].XNA })
	return result
}
