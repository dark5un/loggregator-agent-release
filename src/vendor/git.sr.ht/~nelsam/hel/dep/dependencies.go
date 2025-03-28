package dep

import (
	"context"
	"fmt"
	"go/ast"
	"strings"
	"unicode"
)

// Opt is an option function for modifying a dependency.
type Opt func(Dependency) Dependency

// FromPkg returns an Opt that sets a dependency's package path and name.
func FromPkg(name, path string) Opt {
	return func(d Dependency) Dependency {
		d.pkgPath = path
		d.pkgName = name
		return d
	}
}

// WithImportAlias returns an Opt that sets an import alias for this dependency.
func WithImportAlias(alias string) Opt {
	return func(d Dependency) Dependency {
		d.importAliased = true
		d.pkgName = alias
		return d
	}
}

// WithTypeParams returns an Opt that sets a dependency's type parameters.
func WithTypeParams(params ...*ast.Field) Opt {
	return func(d Dependency) Dependency {
		d.typeParams = params
		return d
	}
}

// Dependency represents a dependency required by other logic. It may be a
// dependency of functions in the production code or a dependency of methods on
// interface types.
type Dependency struct {
	typeName      *ast.Ident
	typeExpr      ast.Expr
	typeParams    []*ast.Field
	pkgName       string
	pkgPath       string
	importAliased bool
}

// New returns a new Dependency.
func New(ctx context.Context, name *ast.Ident, expr ast.Expr, opts ...Opt) Dependency {
	d := Dependency{
		typeName: name,
		typeExpr: expr,
	}
	for _, opt := range opts {
		d = opt(d)
	}
	return d
}

// Name returns d's fully qualified name.
func (d Dependency) Name() string {
	if d.pkgPath == "" {
		return d.typeName.Name
	}
	return fmt.Sprintf("%s.%s", d.pkgPath, d.typeName.Name)
}

// ImportAliased returns whether or not this dependency's import is aliased in
// the local code.
func (d Dependency) ImportAliased() bool {
	return d.importAliased
}

// PkgName returns this dependency's package name.
func (d Dependency) PkgName() string {
	return d.pkgName
}

// PkgPath returns this dependency's package import path.
func (d Dependency) PkgPath() string {
	return d.pkgPath
}

// TypeName returns the local name of this dependency's type.
func (d Dependency) TypeName() string {
	return d.typeName.Name
}

// Type returns this dependency's type expression.
func (d Dependency) Type() ast.Expr {
	return d.typeExpr
}

// Params returns this dependency's type parameters.
func (d Dependency) Params() []*ast.Field {
	return d.typeParams
}

// FromPkg returns a copy of d but from the requested package name and path.
func (d Dependency) FromPkg(ctx context.Context, name, path string) Dependency {
	return FromPkg(name, path)(d)
}

// AvoidCollision returns a copy of this Dependency with a new TypeName that
// avoids collisions with its current TypeName
func (d Dependency) AvoidCollision(ctx context.Context) Dependency {
	titleRunes := []rune(d.PkgName())
	if len(titleRunes) == 0 {
		d.typeName.Name += "_"
		return d
	}
	titleRunes[0] = unicode.ToUpper(titleRunes[0])
	pkgTitle := string(titleRunes)
	if !strings.HasPrefix(d.Name(), pkgTitle) {
		d.typeName.Name = pkgTitle + d.typeName.Name
		return d
	}
	d.typeName.Name += "_"
	return d
}
