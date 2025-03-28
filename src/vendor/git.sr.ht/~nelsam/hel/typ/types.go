// This is free and unencumbered software released into the public
// domain.  For more information, see <http://unlicense.org> or the
// accompanying UNLICENSE file.

package typ

import (
	"context"
	"fmt"
	"go/ast"
	"reflect"
	"regexp"
	"strings"
	"unicode"

	"git.sr.ht/~nelsam/hel/console"
	"git.sr.ht/~nelsam/hel/dep"
	"golang.org/x/tools/go/packages"
)

var (
	// errorMethod is the type of the Error method on error types.
	// It's defined here for any interface types that embed error.
	errorMethod = &ast.Field{
		Names: []*ast.Ident{{Name: "Error"}},
		Type: &ast.FuncType{
			Params: &ast.FieldList{},
			Results: &ast.FieldList{
				List: []*ast.Field{{Type: &ast.Ident{Name: "string"}}},
			},
		},
	}
)

// A GoDir is a type that represents a directory of Go files.
type GoDir interface {
	Path(context.Context) (path string)
	Package(context.Context) *packages.Package
	Import(ctx context.Context, path string) (*packages.Package, error)
}

// Dependency is a type for tracking dependencies of local logic.
type Dependency interface {
	Name() string
	TypeName() string
	PkgPath() string
	PkgName() string
	ImportAliased() bool
	Type() ast.Expr
	Params() []*ast.Field
	AvoidCollision(ctx context.Context) dep.Dependency
	FromPkg(ctx context.Context, name, path string) dep.Dependency
}

// A Dir is a type that represents a directory containing Go
// packages.
type Dir struct {
	dir          string
	pkg          string
	dependencies map[string][]Dependency
	decls        []ast.Decl
}

// Clone returns a copy of d.
func (d *Dir) Clone(ctx context.Context) *Dir {
	copy := *d
	copy.dependencies = make(map[string][]Dependency)
	for k, v := range d.dependencies {
		copy.dependencies[k] = append([]Dependency(nil), v...)
	}
	copy.decls = append([]ast.Decl(nil), d.decls...)
	return &copy
}

// Dir returns the directory path that d represents.
func (d *Dir) Dir(ctx context.Context) string {
	return d.dir
}

// Len returns the number of types that will be returned by
// d.ExportedDecls().
func (d *Dir) Len(ctx context.Context) int {
	return len(d.decls)
}

// Package returns the name of d's importable package.
func (d *Dir) Package(ctx context.Context) string {
	return d.pkg
}

// Decls returns all ast.Decl (declarations) found by d. Interface types with
// anonymous interface types will be flattened, for ease of mocking by other
// logic.
//
// This includes unexported declaratiions.
func (d *Dir) Decls(ctx context.Context) []ast.Decl {
	return d.decls
}

// Dependencies returns all interface types that typ depends on for
// method parameters or results.
func (d *Dir) Dependencies(ctx context.Context, name string) []Dependency {
	added := map[string]struct{}{
		name: {},
	}
	cons := console.FromCtx(ctx)
	cons.Print("dependencies of %v", console.Fmt(name), console.AsVerbose())
	ctx = cons.WithPrefix("- ").Ctx(ctx)
	return d.recurseDeps(ctx, added, name)
}

func (d *Dir) recurseDeps(ctx context.Context, added map[string]struct{}, name string) []Dependency {
	var deps []Dependency
	for _, dep := range d.dependencies[name] {
		if _, ok := added[dep.Name()]; ok {
			continue
		}
		cons := console.FromCtx(ctx)
		cons.Print(dep.Name(), console.AsVerbose())
		added[dep.Name()] = struct{}{}
		deps = append(deps, dep)
		deps = append(deps, d.recurseDeps(cons.WithPrefix("- ").Ctx(ctx), added, dep.Name())...)
	}
	return deps
}

func newDir(ctx context.Context, dirs Dirs, pkg *packages.Package, dir GoDir, pkgName, pkgPath string, unexported bool) *Dir {
	d := &Dir{
		pkg:          pkgName,
		dir:          pkgPath,
		dependencies: make(map[string][]Dependency),
	}
	newDecls, _, depMap := dirs.loadPkgDecls(ctx, pkg, dir, unexported)
	if d.pkg == "" {
		d.pkg = pkg.Name
	}
	for name, deps := range depMap {
		d.dependencies[name] = append(d.dependencies[name], deps...)
	}
	d.decls = append(d.decls, newDecls...)
	return d
}

func matchingSpecs(ctx context.Context, specs []ast.Spec, matchers ...*regexp.Regexp) []ast.Spec {
	var matching []ast.Spec
	for _, s := range specs {
		switch spec := s.(type) {
		case *ast.TypeSpec:
			for _, matcher := range matchers {
				if !matcher.MatchString(spec.Name.Name) {
					continue
				}
				matching = append(matching, spec)
				break
			}
		default:
			continue
		}
	}
	return matching
}

// Filter returns a copy of d with all types filtered. All types in the returned
// Dir will match at least one of the passed in matchers.
func (d *Dir) Filter(ctx context.Context, matchers ...*regexp.Regexp) *Dir {
	copy := d.Clone(ctx)
	copy.decls = nil
	for _, decl := range d.Decls(ctx) {
		switch dt := decl.(type) {
		case *ast.GenDecl:
			newDT := *dt
			newDT.Specs = matchingSpecs(ctx, dt.Specs, matchers...)
			if len(newDT.Specs) > 0 {
				copy.decls = append(copy.decls, &newDT)
			}
		case *ast.FuncDecl:
			for _, matcher := range matchers {
				if !matcher.MatchString(dt.Name.Name) {
					continue
				}
				copy.decls = append(copy.decls, decl)
				break
			}
		}
	}
	return copy
}

// Dirs is a slice of Dir values, to provide sugar for running some
// methods against multiple Dir values.
type Dirs struct {
	dirs []*Dir

	importCache map[string]*Dir
}

// Load loads a Dirs value for goDirs.
func Load(ctx context.Context, unexported bool, goDirs ...GoDir) Dirs {
	d := Dirs{
		importCache: make(map[string]*Dir),
	}
	for _, dir := range goDirs {
		d.dirs = append(d.dirs, newDir(ctx, d, dir.Package(ctx), dir, dir.Package(ctx).Name, dir.Path(ctx), unexported))
	}
	return d
}

// Slice returns the Dir values contained in d in slice form.
func (d Dirs) Slice(ctx context.Context) []*Dir {
	return d.dirs
}

// Filter calls Dir.Filter for each Dir in d.
func (d Dirs) Filter(ctx context.Context, patterns ...string) Dirs {
	if len(patterns) == 0 {
		return d
	}
	matchers := make([]*regexp.Regexp, 0, len(patterns))
	for _, pattern := range patterns {
		matchers = append(matchers, regexp.MustCompile("^"+pattern+"$"))
	}
	newDirs := Dirs{
		importCache: d.importCache,
	}
	for _, dir := range d.dirs {
		copy := dir.Filter(ctx, matchers...)
		if copy.Len(ctx) > 0 {
			newDirs.dirs = append(newDirs.dirs, copy)
		}
	}
	return newDirs
}

func isAny[T, U any](l []T, translate func(T) U, check func(U) bool) bool {
	for _, v := range l {
		if check(translate(v)) {
			return true
		}
	}
	return false
}

type namedDecl struct {
	name string
	decl ast.Decl
}

type depVisitor struct {
	ctx context.Context

	available  []*ast.TypeSpec
	imports    []*ast.ImportSpec
	dir        GoDir
	d          Dirs
	unexported bool

	deps map[string]Dependency
}

func (v *depVisitor) Visit(n ast.Node) ast.Visitor {
	switch n := n.(type) {
	case *ast.FuncDecl:
		if !v.unexported && !unicode.IsUpper([]rune(n.Name.Name)[0]) {
			return v
		}
		cons := console.FromCtx(v.ctx)
		cons.Print("%v [func decl]", console.Fmt(n.Name.Name), console.AsVerbose())
		ctx := cons.WithPrefix("- ").Ctx(v.ctx)
		addSpecs(ctx, v.deps, v.d.loadFieldDependencies(ctx, n.Type.Params, v.available, v.imports, v.dir)...)
		return v
	case *ast.TypeSpec:
		switch t := n.Type.(type) {
		case *ast.InterfaceType:
			if t.Methods == nil {
				return v
			}
			cons := console.FromCtx(v.ctx)
			cons.Print("%v [interface type]", console.Fmt(n.Name.Name), console.AsVerbose())
			ctx := cons.WithPrefix("- ").Ctx(v.ctx)
			for _, meth := range t.Methods.List {
				f, ok := meth.Type.(*ast.FuncType)
				if !ok {
					panic(fmt.Errorf("typ: unexpected non-function type in interface %v", t))
				}
				addSpecs(ctx, v.deps, v.d.loadFieldDependencies(ctx, f.Params, v.available, v.imports, v.dir)...)
				addSpecs(ctx, v.deps, v.d.loadFieldDependencies(ctx, f.Results, v.available, v.imports, v.dir)...)
			}
			return v
		case *ast.StructType:
			if !v.unexported {
				allFields := t.Fields.List
				t.Fields.List = nil
				for _, f := range allFields {
					firstChar := func(n *ast.Ident) rune { return []rune(n.Name)[0] }
					if !isAny(f.Names, firstChar, unicode.IsUpper) {
						continue
					}
					t.Fields.List = append(t.Fields.List, f)
				}
			}
			cons := console.FromCtx(v.ctx)
			cons.Print("%v [struct type]", console.Fmt(n.Name.Name), console.AsVerbose())
			ctx := cons.WithPrefix("- ").Ctx(v.ctx)
			addSpecs(ctx, v.deps, v.d.loadFieldDependencies(ctx, t.Fields, v.available, v.imports, v.dir)...)
			return v
		default:
			return v
		}
	default:
		return v
	}
}

// dependencies returns all types that decl depends on in a testing context.
// This includes parameter/return types for methods on interfaces, parameter
// types for functions, and field types for structs, to name a few.
func (d Dirs) dependencies(ctx context.Context, decl ast.Decl, available []*ast.TypeSpec, withImports []*ast.ImportSpec, dir GoDir, unexported bool) []Dependency {
	v := &depVisitor{
		ctx:        ctx,
		available:  available,
		imports:    withImports,
		dir:        dir,
		d:          d,
		unexported: unexported,
		deps:       make(map[string]Dependency),
	}
	ast.Walk(v, decl)
	depSlice := make([]Dependency, 0, len(v.deps))
	for _, dep := range v.deps {
		depSlice = append(depSlice, dep)
	}
	return depSlice
}

func addSpecs(ctx context.Context, set map[string]Dependency, values ...Dependency) {
	for _, value := range values {
		set[value.Name()] = value
	}
}

func (d Dirs) loadFieldDependencies(ctx context.Context, fields *ast.FieldList, available []*ast.TypeSpec, withImports []*ast.ImportSpec, dir GoDir) (deps []Dependency) {
	if fields == nil {
		return nil
	}
	for _, field := range fields.List {
		deps = append(deps, d.loadTypeDependencies(ctx, field.Type, available, withImports, dir)...)
	}
	return deps
}

func (d Dirs) loadTypeDependencies(ctx context.Context, typ ast.Expr, available []*ast.TypeSpec, withImports []*ast.ImportSpec, dir GoDir) []Dependency {
	switch src := typ.(type) {
	case *ast.ArrayType:
		return d.loadTypeDependencies(ctx, src.Elt, available, withImports, dir)
	case *ast.MapType:
		deps := d.loadTypeDependencies(ctx, src.Key, available, withImports, dir)
		deps = append(deps, d.loadTypeDependencies(ctx, src.Value, available, withImports, dir)...)
		return deps
	case *ast.StarExpr:
		return d.loadTypeDependencies(ctx, src.X, available, withImports, dir)
	case *ast.Ellipsis:
		return d.loadTypeDependencies(ctx, src.Elt, available, withImports, dir)
	case *ast.ChanType:
		return d.loadTypeDependencies(ctx, src.Value, available, withImports, dir)
	case *ast.Ident:
		console.FromCtx(ctx).Print("looking up ident %v", console.Fmt(src.Name), console.AsVerbose())
		spec := findSpec(ctx, src, available)
		console.FromCtx(ctx).Print("found spec %#v", console.Fmt(spec), console.AsVerbose())
		if spec == nil {
			return nil
		}
		if _, ok := spec.Type.(*ast.InterfaceType); !ok {
			return nil
		}
		var opts []dep.Opt
		if spec.TypeParams != nil {
			opts = append(opts, dep.WithTypeParams(spec.TypeParams.List...))
		}
		console.FromCtx(ctx).Print("adding dependency %#v", console.Fmt(spec.Type), console.AsVerbose())
		return []Dependency{dep.New(ctx, spec.Name, spec.Type, opts...)}
	case *ast.SelectorExpr:
		cons := console.FromCtx(ctx)
		selectorName := src.X.(*ast.Ident).String()
		for _, imp := range withImports {
			if imp.Name != nil && imp.Name.Name != selectorName {
				continue
			}
			importPath := strings.Trim(imp.Path.Value, `"`)
			cons.Print("looking up dependency %s.%s in import %v", console.Fmt(selectorName, src.Sel.String(), importPath), console.AsVerbose())
			cons := cons.WithPrefix("- ")
			ctx := cons.Ctx(ctx)
			impDir, ok := d.importCache[importPath]
			if !ok {
				pkg, err := dir.Import(ctx, importPath)
				if err != nil {
					continue
				}
				if imp.Name == nil && pkg.Name != selectorName {
					// The import has been cached, but we don't need to process
					// the types just yet.
					continue
				}
				impDir = newDir(ctx, d, pkg, dir, "", "", false)
				d.importCache[importPath] = impDir
			}
			opts := []dep.Opt{dep.FromPkg(impDir.pkg, importPath)}
			importName := impDir.pkg
			if imp.Name != nil {
				importName = imp.Name.Name
				opts = append(opts, dep.WithImportAlias(imp.Name.Name))
			}
			for _, decl := range impDir.Decls(ctx) {
				switch dt := decl.(type) {
				case *ast.GenDecl:
					for _, spec := range dt.Specs {
						switch st := spec.(type) {
						case *ast.TypeSpec:
							if st.Name.Name != src.Sel.String() {
								continue
							}
							if st.TypeParams != nil {
								opts = append(opts, dep.WithTypeParams(st.TypeParams.List...))
							}
							dependencies := []Dependency{dep.New(ctx, st.Name, st.Type, opts...)}
							switch st.Type.(type) {
							case *ast.InterfaceType:
								// Only interface types need to recurse to sub-dependencies,
								// because concrete sub-dependencies won't be called in a
								// way that allows mocks to be used. Non-interface types are
								// really only added to the dependency list so we know what
								// to add to the import clause.
								cons.Print("%s [dependency]", console.Fmt(st.Name.Name), console.AsVerbose())
								cons := cons.WithPrefix("- ")
								ctx := cons.Ctx(ctx)
								subDeps := impDir.Dependencies(ctx, st.Name.Name)
								for i, d := range subDeps {
									cons.Print("checking subdep %v", console.Fmt(d.Name()), console.AsVerbose())
									if d.PkgPath() != "" {
										continue
									}
									subDeps[i] = d.FromPkg(ctx, importName, importPath)
								}
								return append(dependencies, subDeps...)
							default:
								return dependencies
							}
						default:
							continue
						}
					}
				case *ast.FuncDecl:
					if dt.Name.Name != src.Sel.String() {
						continue
					}
					if dt.Type.TypeParams != nil {
						opts = append(opts, dep.WithTypeParams(dt.Type.TypeParams.List...))
					}
					return []Dependency{dep.New(ctx, dt.Name, dt.Type, opts...)}
				}
			}
		}
		return nil
	case *ast.FuncType:
		return d.loadFieldDependencies(ctx, src.Params, available, withImports, dir)
	default:
		return nil
	}
}

func (d Dirs) loadPkgDecls(ctx context.Context, pkg *packages.Package, dir GoDir, unexported bool) (decls []ast.Decl, imports map[string][]*ast.ImportSpec, depMap map[string][]Dependency) {
	// TODO: if/when performance starts getting hairy, this is a good place to
	// skip unexported symbols. Unexported interface types should probably still
	// be processed since they _technically_ can be used by exported logic, but
	// unexported functions and the like could be skipped.
	depMap = make(map[string][]Dependency)
	imports = make(map[string][]*ast.ImportSpec)
	var types []*ast.TypeSpec
	defer func() {
		for _, decl := range decls {
			switch decl := decl.(type) {
			case *ast.GenDecl:
				for _, spec := range decl.Specs {
					switch spec := spec.(type) {
					case *ast.TypeSpec:
						cons := console.FromCtx(ctx)
						cons.Print("%v [type decl]", console.Fmt(spec.Name.Name), console.AsVerbose())
						ctx := cons.WithPrefix("- ").Ctx(ctx)
						depMap[spec.Name.Name] = append(depMap[spec.Name.Name], d.dependencies(ctx, decl, types, imports[spec.Name.Name], dir, unexported)...)
					default:
						continue
					}
				}
			case *ast.FuncDecl:
				name := decl.Name.Name
				if decl.Recv != nil && len(decl.Recv.List) > 0 {
					ident, ok := decl.Recv.List[0].Type.(*ast.Ident)
					if ok {
						name = ident.Name
					}
				}
				cons := console.FromCtx(ctx)
				cons.Print("%v [func decl]", console.Fmt(name), console.AsVerbose())
				ctx := cons.WithPrefix("- ").Ctx(ctx)
				depMap[name] = append(depMap[name], d.dependencies(ctx, decl, types, imports[decl.Name.Name], dir, unexported)...)
				continue
			default:
				continue
			}
		}
	}()
	for _, f := range pkg.Syntax {
		fileImports := f.Imports
		fileDecls := loadFileDecls(ctx, f)
		var fileTypes []*ast.TypeSpec
		for _, decl := range fileDecls {
			switch decl := decl.(type) {
			case *ast.GenDecl:
				for _, spec := range decl.Specs {
					typ, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}
					imports[typ.Name.Name] = fileImports
					fileTypes = append(fileTypes, typ)
				}
			case *ast.FuncDecl:
				imports[decl.Name.Name] = fileImports
			default:
				continue
			}
		}

		// flattenAnon needs to be called for each file, but the
		// withSpecs parameter needs *all* specs, from *all* files.
		// So we defer the flatten call until all files are processed.
		defer func() {
			d.flattenAnon(ctx, fileTypes, types, fileImports, imports, dir, unexported)
		}()

		decls = append(decls, fileDecls...)
		types = append(types, fileTypes...)
	}
	return decls, imports, depMap
}

func loadFileDecls(ctx context.Context, f *ast.File) (decls []ast.Decl) {
	for _, d := range f.Decls {
		switch decl := d.(type) {
		case *ast.GenDecl:
			// This consists of both concrete and interface types.
			decls = append(decls, decl)
		case *ast.FuncDecl:
			decls = append(decls, decl)
		default:
			continue
		}
	}
	return decls
}

func (d Dirs) flattenAnon(ctx context.Context, specs, withSpecs []*ast.TypeSpec, fileImports []*ast.ImportSpec, declImports map[string][]*ast.ImportSpec, dir GoDir, unexported bool) {
	for _, spec := range specs {
		d.flatten(ctx, spec, withSpecs, fileImports, declImports, dir, unexported)
	}
}

func (d Dirs) flatten(ctx context.Context, spec *ast.TypeSpec, withSpecs []*ast.TypeSpec, fileImports []*ast.ImportSpec, declImports map[string][]*ast.ImportSpec, dir GoDir, unexported bool) {
	switch t := spec.Type.(type) {
	case *ast.InterfaceType:
		if t.Methods == nil {
			return
		}
		used := make(map[string]struct{})
		methods := make([]*ast.Field, 0, len(t.Methods.List))
		for _, method := range t.Methods.List {
			switch src := method.Type.(type) {
			case *ast.FuncType:
				if _, ok := used[method.Names[0].Name]; ok {
					for i, m := range methods {
						if m.Names[0].Name != method.Names[0].Name {
							continue
						}
						methods[i] = method
						break
					}
					continue
				}
				methods = append(methods, method)
				used[method.Names[0].Name] = struct{}{}
			case *ast.Ident:
				for _, m := range d.findAnonMethods(ctx, src, withSpecs, declImports[src.Name], declImports, dir, unexported) {
					if _, ok := used[m.Names[0].Name]; ok {
						continue
					}
					methods = append(methods, m)
					used[m.Names[0].Name] = struct{}{}
				}
			case *ast.SelectorExpr:
				importedTypes, imports, _ := d.findImportedTypes(ctx, src.X.(*ast.Ident), fileImports, dir)
				for _, m := range d.findAnonMethods(ctx, src.Sel, importedTypes, imports[src.Sel.Name], imports, dir, unexported) {
					if _, ok := used[m.Names[0].Name]; ok {
						continue
					}
					methods = append(methods, m)
					used[m.Names[0].Name] = struct{}{}
				}
			}
		}
		t.Methods.List = methods
	case *ast.Ident:
		// type alias in the same package
		realSpec := findSpec(ctx, t, withSpecs)
		if realSpec == nil {
			return
		}
		spec.Type = realSpec.Type
		d.flatten(ctx, spec, withSpecs, declImports[t.Name], declImports, dir, unexported)
	case *ast.SelectorExpr:
		// type alias in imported package
		imported, imports, _ := d.findImportedTypes(ctx, t.X.(*ast.Ident), fileImports, dir)
		realSpec := findSpec(ctx, t.Sel, imported)
		if realSpec == nil {
			return
		}
		spec.Type = realSpec.Type
		d.flatten(ctx, spec, imported, imports[realSpec.Name.Name], imports, dir, unexported)
	}
}

func (d Dirs) findImportedTypes(ctx context.Context, name *ast.Ident, withImports []*ast.ImportSpec, dir GoDir) ([]*ast.TypeSpec, map[string][]*ast.ImportSpec, map[string][]Dependency) {
	importName := name.String()
	cons := console.FromCtx(ctx)
	for _, imp := range withImports {
		path := strings.Trim(imp.Path.Value, `"`)
		pkg, err := dir.Import(ctx, path)
		if err != nil {
			cons.Print("skipping import %q: %v", console.Fmt(path, err), console.Once(), console.NewlinePfx())
			continue
		}
		name := pkg.Name
		if imp.Name != nil {
			name = imp.Name.String()
		}
		if name != importName {
			continue
		}
		cons.Print("%v [package]", console.Fmt(pkg), console.AsVerbose())
		ctx := cons.WithPrefix("- ").Ctx(ctx)
		decls, imports, deps := d.loadPkgDecls(ctx, pkg, dir, false)
		var types []*ast.TypeSpec
		for _, d := range decls {
			switch decl := d.(type) {
			case *ast.GenDecl:
				for _, s := range decl.Specs {
					typ, ok := s.(*ast.TypeSpec)
					if !ok {
						continue
					}
					types = append(types, copy(typ))
				}
			case *ast.FuncDecl:
				// TODO: recursive function/type dependencies?
			default:
				continue
			}
		}
		addSelector(ctx, types, importName)
		return types, imports, deps
	}
	return nil, nil, nil
}

func addSelector(ctx context.Context, typs []*ast.TypeSpec, selector string) {
	for _, typ := range typs {
		inter, ok := typ.Type.(*ast.InterfaceType)
		if !ok {
			continue
		}
		for _, meth := range inter.Methods.List {
			addFuncSelectors(ctx, meth.Type.(*ast.FuncType), selector)
		}
	}
}

func addFuncSelectors(ctx context.Context, method *ast.FuncType, selector string) {
	if method.Params != nil {
		addFieldSelectors(ctx, method.Params.List, selector)
	}
	if method.Results != nil {
		addFieldSelectors(ctx, method.Results.List, selector)
	}
}

func addFieldSelectors(ctx context.Context, fields []*ast.Field, selector string) {
	for idx, field := range fields {
		fields[idx] = addFieldSelector(ctx, field, selector)
	}
}

func addFieldSelector(ctx context.Context, field *ast.Field, selector string) *ast.Field {
	switch src := field.Type.(type) {
	case *ast.Ident:
		if !unicode.IsUpper(rune(src.String()[0])) {
			return field
		}
		return &ast.Field{
			Type: &ast.SelectorExpr{
				X:   &ast.Ident{Name: selector},
				Sel: src,
			},
		}
	case *ast.FuncType:
		addFuncSelectors(ctx, src, selector)
	}
	return field
}

func findSpec(ctx context.Context, ident *ast.Ident, inSpecs []*ast.TypeSpec) *ast.TypeSpec {
	for _, spec := range inSpecs {
		if spec.Name.String() == ident.Name {
			return spec
		}
	}
	return nil
}

func (d Dirs) findAnonMethods(
	ctx context.Context,
	ident *ast.Ident,
	withSpecs []*ast.TypeSpec,
	fileImports []*ast.ImportSpec,
	declImports map[string][]*ast.ImportSpec,
	dir GoDir,
	unexported bool,
) []*ast.Field {
	spec := findSpec(ctx, ident, withSpecs)
	if spec == nil {
		if ident.Name != "error" {
			// TODO: do something nicer with this error.
			panic(fmt.Errorf("Can't find anonymous type %s", ident.Name))
		}
		return []*ast.Field{errorMethod}
	}
	d.flatten(ctx, spec, withSpecs, fileImports, declImports, dir, unexported)
	anon := spec.Type.(*ast.InterfaceType)
	return anon.Methods.List
}

func copy[T any](v T) T {
	var cpy T
	copyValue(make(map[uintptr]reflect.Value), reflect.ValueOf(&cpy), reflect.ValueOf(&v))
	return cpy
}

func copyValue(addrMap map[uintptr]reflect.Value, dstV, srcV reflect.Value) {
	for {
		switch reflect.Ptr {
		case dstV.Kind(), srcV.Kind():
			if srcV.IsNil() {
				return
			}
			addr := srcV.Pointer()
			if v, ok := addrMap[addr]; ok {
				dstV.Set(v)
				return
			}
			if dstV.IsNil() {
				dstV.Set(reflect.New(dstV.Type().Elem()))
			}
			addrMap[addr] = dstV
			dstV = dstV.Elem()
			srcV = srcV.Elem()
			continue
		}
		break
	}
	if dstV.Kind() != srcV.Kind() {
		panic(fmt.Errorf("unmatched kinds %v and %v", dstV.Kind(), srcV.Kind()))
	}
	switch dstV.Kind() {
	case reflect.Interface:
		if srcV.IsNil() {
			return
		}
		cpy := reflect.New(srcV.Elem().Type()).Elem()
		copyValue(addrMap, cpy, srcV.Elem())
		dstV.Set(cpy)
	case reflect.Struct:
		for i := 0; i < dstV.NumField(); i++ {
			copyValue(addrMap, dstV.Field(i), srcV.Field(i))
		}
	case reflect.Slice:
		for i := 0; i < srcV.Len(); i++ {
			newV := reflect.New(dstV.Type().Elem()).Elem()
			copyValue(addrMap, newV, srcV.Index(i))
			dstV.Set(reflect.Append(dstV, newV))
		}
	case reflect.Map:
		if dstV.IsNil() {
			dstV.Set(reflect.MakeMap(dstV.Type()))
		}
		for _, kv := range srcV.MapKeys() {
			newV := reflect.New(srcV.Type().Elem()).Elem()
			copyValue(addrMap, newV, srcV.MapIndex(kv))
			dstV.SetMapIndex(kv, newV)
		}
	default:
		dstV.Set(srcV)
	}
}
