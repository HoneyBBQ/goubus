package main

import (
	"go/types"
	"slices"
)

// findLeaks records every type that the exported declaration text exposes and
// a caller cannot name: an unexported type of the package, or a type of an
// internal package. A caller can hold a value of such a type and call its
// methods, yet cannot declare a variable of it or find it in the reports, so
// it is part of the API that no report lists.
func (r *renderer) findLeaks(text string, t types.Type) {
	w := &leakWalker{r: r, text: text, seen: map[types.Type]bool{}}
	w.walk(t)
}

// leakWalker visits the types a declaration is built from.
type leakWalker struct {
	r    *renderer
	seen map[types.Type]bool
	text string
}

func (w *leakWalker) walk(t types.Type) {
	if t == nil || w.seen[t] {
		return
	}
	w.seen[t] = true
	switch t := t.(type) {
	case *types.Named:
		w.check(t.Obj())
		for arg := range t.TypeArgs().Types() {
			w.walk(arg)
		}
	case *types.Alias:
		// An alias is a name of its own: whatever it stands for is exposed on
		// purpose, and its line in the report shows the target.
		w.check(t.Obj())
	case *types.Map:
		w.walk(t.Key())
		w.walk(t.Elem())
	case interface{ Elem() types.Type }: // a pointer, slice, array or channel
		w.walk(t.Elem())
	case *types.Signature:
		w.walkTuple(t.Params())
		w.walkTuple(t.Results())
	case *types.Struct:
		for f := range t.Fields() {
			if f.Exported() {
				w.walk(f.Type())
			}
		}
	case *types.Interface:
		for m := range t.ExplicitMethods() {
			if m.Exported() {
				w.walk(m.Type())
			}
		}
	}
}

func (w *leakWalker) walkTuple(tuple *types.Tuple) {
	for v := range tuple.Variables() {
		w.walk(v.Type())
	}
}

func (w *leakWalker) check(obj *types.TypeName) {
	if obj.Pkg() == nil {
		return
	}
	var what string
	switch {
	case isInternal(obj.Pkg().Path()):
		what = "the internal type " + obj.Pkg().Path() + "." + obj.Name()
	case obj.Pkg().Path() == w.r.pkg.Path() && !obj.Exported():
		what = "the unexported type " + obj.Name()
	default:
		return
	}
	leak := w.text + " exposes " + what
	if !slices.Contains(w.r.leaks, leak) {
		w.r.leaks = append(w.r.leaks, leak)
	}
}
