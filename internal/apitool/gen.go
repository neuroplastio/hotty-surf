package main

import (
	"bufio"
	"flag"
	"fmt"
	"go/types"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"golang.org/x/tools/go/packages"
)

func genCmd(args []string) error {
	fs := flag.NewFlagSet("gen", flag.ExitOnError)
	out := fs.String("o", "", "write here (default stdout)")
	fs.Parse(args)
	lines, err := generate(fs.Args())
	if err != nil {
		return err
	}
	return writeLines(*out, lines)
}

// generate is the exported API of the modules in dirs, sorted.
func generate(dirs []string) ([]string, error) {
	var lines []string
	for _, d := range dirs {
		cfg := &packages.Config{Mode: packages.NeedName | packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo, Dir: d}
		pkgs, err := packages.Load(cfg, "./...")
		if err != nil {
			return nil, err
		}
		for _, p := range pkgs {
			if len(p.Errors) > 0 {
				return nil, fmt.Errorf("%s: %v", p.PkgPath, p.Errors[0])
			}
			if p.Name == "main" || isInternal(p.PkgPath) || strings.Contains(p.PkgPath, "/examples/") {
				continue
			}
			lines = append(lines, pkgLines(p.Types)...)
		}
	}
	slices.Sort(lines)
	return slices.Compact(lines), nil
}

func isInternal(path string) bool {
	for _, e := range strings.Split(path, "/") {
		if e == "internal" {
			return true
		}
	}
	return false
}

func pkgLines(pkg *types.Package) []string {
	q := func(p *types.Package) string {
		if p == pkg {
			return ""
		}
		return p.Name()
	}
	pre := "pkg " + pkg.Path() + ", "
	var out []string
	add := func(s string) { out = append(out, pre+s) }
	sc := pkg.Scope()
	for _, name := range sc.Names() {
		obj := sc.Lookup(name)
		if !obj.Exported() {
			continue
		}
		switch o := obj.(type) {
		case *types.Func:
			add("func " + name + sig(o.Type().(*types.Signature), q, true))
		case *types.Var:
			add("var " + name + " " + types.TypeString(o.Type(), q))
		case *types.Const:
			add(fmt.Sprintf("const %s %s = %s", name, types.TypeString(o.Type(), q), o.Val().ExactString()))
		case *types.TypeName:
			out = append(out, typeLines(pre, o, q)...)
		}
	}
	return out
}

func typeLines(pre string, o *types.TypeName, q types.Qualifier) []string {
	name := o.Name()
	var out []string
	if o.IsAlias() {
		return []string{pre + "type " + name + " = " + types.TypeString(types.Unalias(o.Type()), q)}
	}
	named := o.Type().(*types.Named)
	tparams := ""
	if tp := named.TypeParams(); tp != nil && tp.Len() > 0 {
		tparams = tparamString(tp, q)
	}
	switch u := named.Underlying().(type) {
	case *types.Struct:
		out = append(out, pre+"type "+name+tparams+" struct")
		for i := range u.NumFields() {
			f := u.Field(i)
			if f.Exported() {
				fname := f.Name()
				if f.Embedded() {
					fname = "embedded"
				}
				out = append(out, pre+"type "+name+" struct, "+fname+" "+types.TypeString(strip(f.Type()), q))
			}
		}
	case *types.Interface:
		var ms []string
		for i := range u.NumExplicitMethods() {
			m := u.ExplicitMethod(i)
			if m.Exported() {
				ms = append(ms, m.Name())
				out = append(out, pre+"type "+name+" interface, "+m.Name()+sig(m.Type().(*types.Signature), q, false))
			}
		}
		out = append(out, pre+"type "+name+tparams+" interface { "+strings.Join(ms, ", ")+" }")
	default:
		out = append(out, pre+"type "+name+tparams+" "+types.TypeString(u, q))
	}
	// Methods, on T and *T.
	for i := range named.NumMethods() {
		m := named.Method(i)
		if !m.Exported() {
			continue
		}
		s := m.Type().(*types.Signature)
		recv := name
		if _, ok := s.Recv().Type().(*types.Pointer); ok {
			recv = "*" + name
		}
		out = append(out, pre+"method ("+recv+") "+m.Name()+sig(s, q, false))
	}
	return out
}

func tparamString(tp *types.TypeParamList, q types.Qualifier) string {
	var b []string
	for i := range tp.Len() {
		p := tp.At(i)
		b = append(b, p.Obj().Name()+" "+types.TypeString(p.Constraint(), q))
	}
	return "[" + strings.Join(b, ", ") + "]"
}

// sig is a signature without parameter names: (T, U) R.
func sig(s *types.Signature, q types.Qualifier, withTParams bool) string {
	var b strings.Builder
	if withTParams && s.TypeParams() != nil && s.TypeParams().Len() > 0 {
		b.WriteString(tparamString(s.TypeParams(), q))
	}
	b.WriteString("(")
	ps := s.Params()
	for i := range ps.Len() {
		if i > 0 {
			b.WriteString(", ")
		}
		t := ps.At(i).Type()
		if s.Variadic() && i == ps.Len()-1 {
			b.WriteString("..." + types.TypeString(strip(t.(*types.Slice).Elem()), q))
		} else {
			b.WriteString(types.TypeString(strip(t), q))
		}
	}
	b.WriteString(")")
	rs := s.Results()
	switch rs.Len() {
	case 0:
	case 1:
		b.WriteString(" " + types.TypeString(strip(rs.At(0).Type()), q))
	default:
		var r []string
		for i := range rs.Len() {
			r = append(r, types.TypeString(strip(rs.At(i).Type()), q))
		}
		b.WriteString(" (" + strings.Join(r, ", ") + ")")
	}
	return b.String()
}

func readLines(path string) ([]string, error) {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		l := strings.TrimSpace(sc.Text())
		if l != "" && !strings.HasPrefix(l, "#") {
			out = append(out, l)
		}
	}
	return out, sc.Err()
}

func writeLines(path string, lines []string) error {
	s := strings.Join(lines, "\n")
	if len(lines) > 0 {
		s += "\n"
	}
	if path == "" {
		_, err := os.Stdout.WriteString(s)
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(s), 0o644)
}

// strip is t with the parameter names of the function types in it left
// out, so that renaming a parameter changes no line.
func strip(t types.Type) types.Type {
	switch u := t.(type) {
	case *types.Signature:
		if u.TypeParams() != nil && u.TypeParams().Len() > 0 {
			return t
		}
		return types.NewSignatureType(nil, nil, nil, stripTuple(u.Params()), stripTuple(u.Results()), u.Variadic())
	case *types.Pointer:
		return types.NewPointer(strip(u.Elem()))
	case *types.Slice:
		return types.NewSlice(strip(u.Elem()))
	case *types.Map:
		return types.NewMap(strip(u.Key()), strip(u.Elem()))
	case *types.Chan:
		return types.NewChan(u.Dir(), strip(u.Elem()))
	}
	return t
}

func stripTuple(t *types.Tuple) *types.Tuple {
	vs := make([]*types.Var, t.Len())
	for i := range t.Len() {
		vs[i] = types.NewParam(0, nil, "", strip(t.At(i).Type()))
	}
	return types.NewTuple(vs...)
}
