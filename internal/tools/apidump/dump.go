package main

import (
	"cmp"
	"fmt"
	"go/constant"
	"go/types"
	"math"
	"slices"
	"strconv"
	"strings"
)

// Kinds of top-level object, in the order they appear in a report.
const (
	kindConst = iota
	kindVar
	kindFunc
	kindType
)

// Ranks of the lines that belong to one type, in the order they appear: the
// declaration, then what makes it a closed interface or a constraint, then
// the struct fields in declaration order, then the methods by name.
const (
	rankDecl = iota
	rankSealed
	rankConstraint
	rankField
	rankMethod
)

// entry is one line of a report with the key it sorts by.
type entry struct {
	text string
	name string
	sub  string
	kind int
	rank int
}

// renderer turns the exported objects of one package into report lines.
type renderer struct {
	pkg *types.Package
	// names maps the short name of every imported package a line mentions to
	// its path, so two packages that share a name are told apart.
	names map[string]string
	// leaks lists the types an exported identifier exposes that a caller
	// cannot name: unexported ones, and those of internal packages.
	leaks   []string
	entries []entry
}

// render returns the report of pkg: one line per exported object, sorted, and
// the leaks it found.
func render(pkg *types.Package) ([]string, []string) {
	r := &renderer{pkg: pkg, names: map[string]string{}}
	scope := pkg.Scope()
	for _, name := range scope.Names() {
		obj := scope.Lookup(name)
		if !obj.Exported() {
			continue
		}
		switch obj := obj.(type) {
		case *types.Const:
			r.constant(obj)
		case *types.Var:
			r.variable(obj)
		case *types.Func:
			r.function(obj)
		case *types.TypeName:
			r.typeName(obj)
		}
	}
	slices.SortStableFunc(r.entries, func(a, b entry) int {
		return cmp.Or(
			cmp.Compare(a.kind, b.kind),
			cmp.Compare(a.name, b.name),
			cmp.Compare(a.rank, b.rank),
			cmp.Compare(a.sub, b.sub),
		)
	})
	lines := make([]string, len(r.entries))
	for i, e := range r.entries {
		lines[i] = e.text
	}

	return lines, r.leaks
}

func (r *renderer) variable(v *types.Var) {
	r.add(entry{kind: kindVar, name: v.Name(), text: "var " + v.Name() + " " + r.typeString(v.Type())}, v.Type())
}

func (r *renderer) add(e entry, exposed ...types.Type) {
	r.entries = append(r.entries, e)
	for _, t := range exposed {
		r.findLeaks(e.text, t)
	}
}

func (r *renderer) constant(c *types.Const) {
	r.add(entry{
		kind: kindConst,
		name: c.Name(),
		text: "const " + c.Name() + " " + r.typeString(c.Type()) + " = " + constantString(c.Val()),
	})
}

// constantString renders a constant exactly: a changed value is a break, so
// nothing is rounded or shortened.
func constantString(v constant.Value) string {
	switch v.Kind() {
	case constant.String:
		return strconv.Quote(constant.StringVal(v))
	case constant.Float:
		// The shortest decimal that reads back as the same float64: what the
		// constant becomes the moment it is used.
		if f, _ := constant.Float64Val(v); !math.IsInf(f, 0) {
			return strconv.FormatFloat(f, 'g', -1, 64)
		}
	case constant.Unknown, constant.Bool, constant.Int, constant.Complex:
	}

	return v.ExactString()
}

func (r *renderer) function(f *types.Func) {
	sig, ok := f.Type().(*types.Signature)
	if !ok {
		return
	}
	r.add(entry{kind: kindFunc, name: f.Name(), text: "func " + f.Name() + r.signature(sig)}, sig)
}

// signature renders a signature without its leading "func", so a name can be
// put in front of it.
func (r *renderer) signature(sig *types.Signature) string {
	return strings.TrimPrefix(r.typeString(sig), "func")
}

func (r *renderer) typeString(t types.Type) string {
	return types.TypeString(t, r.qualifier)
}

// qualifier names the package of a type the way a caller reads it: by name,
// by path where two imports share a name, and by path for internal packages
// so that a leak cannot be missed.
func (r *renderer) qualifier(p *types.Package) string {
	switch {
	case p.Path() == r.pkg.Path():
		return ""
	case isInternal(p.Path()):
		return p.Path()
	}
	if path, seen := r.names[p.Name()]; seen && path != p.Path() {
		return p.Path()
	}
	r.names[p.Name()] = p.Path()

	return p.Name()
}

func isInternal(path string) bool {
	return strings.HasSuffix(path, "/internal") || strings.Contains(path, "/internal/")
}

func (r *renderer) typeName(obj *types.TypeName) {
	name := obj.Name()
	if alias, ok := obj.Type().(*types.Alias); ok {
		r.alias(name, alias)

		return
	}
	named, ok := obj.Type().(*types.Named)
	if !ok {
		return
	}
	r.add(entry{
		kind: kindType, name: name,
		text: "type " + name + r.typeParams(named.TypeParams()) + " " + r.underlying(named.Underlying()),
	})
	r.members(name, named)
}

// members lists what a caller can reach through a defined type.
func (r *renderer) members(name string, named *types.Named) {
	switch u := named.Underlying().(type) {
	case *types.Struct:
		r.structFields(name, u, new(int))
	case *types.Interface:
		r.interfaceMembers(name, u)

		return
	}
	r.methods(name, named)
}

// alias records an alias and, when it names a type of an internal package,
// that type's members too: callers can use them through the alias, and no
// report of that package lists them.
func (r *renderer) alias(name string, alias *types.Alias) {
	r.add(entry{
		kind: kindType, name: name,
		text: "type " + name + r.typeParams(alias.TypeParams()) + " = " + r.typeString(alias.Rhs()),
	})
	named, ok := types.Unalias(alias).(*types.Named)
	if !ok || named.Obj().Pkg() == nil || !isInternal(named.Obj().Pkg().Path()) {
		return
	}
	r.members(name, named)
}

// underlying renders what a defined type is made of. Structs and interfaces
// are only named, because their members get lines of their own.
func (r *renderer) underlying(t types.Type) string {
	switch t.(type) {
	case *types.Struct:
		return "struct"
	case *types.Interface:
		return "interface"
	}

	return r.typeString(t)
}

func (r *renderer) typeParams(list *types.TypeParamList) string {
	if list == nil || list.Len() == 0 {
		return ""
	}
	params := make([]string, 0, list.Len())
	for tp := range list.TypeParams() {
		params = append(params, tp.Obj().Name()+" "+r.typeString(tp.Constraint()))
	}

	return "[" + strings.Join(params, ", ") + "]"
}

// structFields lists the exported fields of a struct in declaration order,
// because a reordering breaks unkeyed literals. The fields an unexported
// embedded struct promotes are listed as the struct's own.
func (r *renderer) structFields(owner string, st *types.Struct, index *int) {
	for i := range st.NumFields() {
		f := st.Field(i)
		if f.Embedded() && !f.Exported() {
			if inner, ok := embeddedStruct(f.Type()); ok {
				r.structFields(owner, inner, index)
			}

			continue
		}
		if !f.Exported() {
			continue
		}
		*index++
		text := "field " + owner + "." + f.Name() + " " + r.typeString(f.Type())
		if f.Embedded() {
			text = "embed " + owner + " " + r.typeString(f.Type())
		}
		if tag := st.Tag(i); tag != "" {
			text += " " + strconv.Quote(tag)
		}
		r.add(entry{kind: kindType, name: owner, rank: rankField, sub: fmt.Sprintf("%06d", *index), text: text}, f.Type())
	}
}

func embeddedStruct(t types.Type) (*types.Struct, bool) {
	t = types.Unalias(t)
	if p, ok := t.(*types.Pointer); ok {
		t = types.Unalias(p.Elem())
	}
	st, ok := t.Underlying().(*types.Struct)

	return st, ok
}

// interfaceMembers lists the methods of an interface, those it embeds from
// other interfaces included, and what narrows it: a type set makes it a
// constraint, and an unexported method means no other package can implement
// it, so that methods can be added to it without a break.
func (r *renderer) interfaceMembers(owner string, iface *types.Interface) {
	for m := range iface.Methods() {
		if !m.Exported() {
			r.add(entry{kind: kindType, name: owner, rank: rankSealed, text: "sealed " + owner})

			continue
		}
		sig, ok := m.Type().(*types.Signature)
		if !ok {
			continue
		}
		r.add(entry{
			kind: kindType, name: owner, rank: rankMethod, sub: m.Name(),
			text: "method (" + owner + ") " + m.Name() + r.signature(sig),
		}, sig)
	}
	for embedded := range iface.EmbeddedTypes() {
		if _, plain := embedded.Underlying().(*types.Interface); plain && !hasTypeSet(embedded) {
			continue
		}
		text := "constraint " + owner + " " + r.typeString(embedded)
		r.add(entry{kind: kindType, name: owner, rank: rankConstraint, sub: text, text: text})
	}
}

// hasTypeSet reports whether t restricts the types that satisfy it, as a
// union or a ~T term does, rather than only requiring methods.
func hasTypeSet(t types.Type) bool {
	switch t := t.(type) {
	case *types.Union:
		return true
	case *types.Named:
		iface, ok := t.Underlying().(*types.Interface)

		return ok && !iface.IsMethodSet()
	}

	return false
}

// methods lists the exported methods of a defined type that is not an
// interface: its own and those promoted from embedded fields, each with the
// receiver a caller needs (a pointer method is not in the method set of the
// value).
func (r *renderer) methods(owner string, named *types.Named) {
	pointers := types.NewMethodSet(types.NewPointer(named))
	values := types.NewMethodSet(named)
	typeParams := ""
	if list := named.TypeParams(); list != nil && list.Len() > 0 {
		names := make([]string, 0, list.Len())
		for tp := range list.TypeParams() {
			names = append(names, tp.Obj().Name())
		}
		typeParams = "[" + strings.Join(names, ", ") + "]"
	}
	for sel := range pointers.Methods() {
		m := sel.Obj()
		if !m.Exported() {
			continue
		}
		sig, ok := m.Type().(*types.Signature)
		if !ok {
			continue
		}
		receiver := "*" + owner + typeParams
		if values.Lookup(m.Pkg(), m.Name()) != nil {
			receiver = owner + typeParams
		}
		r.add(entry{
			kind: kindType, name: owner, rank: rankMethod, sub: m.Name(),
			text: "method (" + receiver + ") " + m.Name() + r.signature(sig),
		}, sig)
	}
}
