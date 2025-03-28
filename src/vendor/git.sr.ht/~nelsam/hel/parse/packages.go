// This is free and unencumbered software released into the public
// domain.  For more information, see <http://unlicense.org> or the
// accompanying UNLICENSE file.

package parse

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
)

var (
	cwd       string
	gopathEnv = os.Getenv("GOPATH")
	gopath    = strings.Split(gopathEnv, string(os.PathListSeparator))
)

func init() {
	var err error
	cwd, err = os.Getwd()
	if err != nil {
		panic(err)
	}
}

// Dir represents a parsed directory containing go files.
type Dir struct {
	pkg         *packages.Package
	fsPath      string
	importCache map[string]*packages.Package
}

// Packages looks for directories matching the passed in package patterns and
// returns Dir values for each directory that can be successfully imported and
// is found to match one of the patterns.
func Packages(ctx context.Context, pkgPatterns ...string) ([]Dir, error) {
	return load(ctx, cwd, pkgPatterns...)
}

func load(_ context.Context, fromDir string, pkgPatterns ...string) (dirs []Dir, _ error) {
	// All Dir values share the same import cache.
	impCache := make(map[string]*packages.Package)

	// TODO: local imports (./somepkg, etc)

	// NOTE: we used to use NeedDeps here, which would parse dependencies
	// recursively. That became _excruciatingly slow_ with some packages. It's
	// much faster to lazy-load dependencies as needed (and cache them in a
	// shared import cache).
	pkgs, err := packages.Load(&packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedImports | packages.NeedSyntax,
	}, pkgPatterns...)
	if err != nil {
		return nil, fmt.Errorf("failed to load package patterns %v: %w\n[hint: ensure that all code compiles, including test code]", pkgPatterns, err)
	}
	for _, pkg := range pkgs {
		fsPath := ""
		if len(pkg.GoFiles) > 0 {
			fsPath = filepath.Dir(pkg.GoFiles[0])
		}
		dirs = append(dirs, Dir{pkg: pkg, fsPath: fsPath, importCache: impCache})
	}
	return dirs, nil
}

// Path returns the file path to d.
func (d Dir) Path(ctx context.Context) string {
	return d.fsPath
}

// Package returns the *packages.Package for d
func (d Dir) Package(ctx context.Context) *packages.Package {
	return d.pkg
}

// Import imports path from srcDir, then loads the ast for that package.
// It ensures that the returned ast is for the package that would be
// imported by an import clause.
//
// Import returns an error if the package requested has no go files in it.
func (d Dir) Import(ctx context.Context, path string) (*packages.Package, error) {
	if path == "" {
		return nil, fmt.Errorf("cannot import an empty package path")
	}
	if pkg, ok := d.importCache[path]; ok {
		return pkg, nil
	}
	pkgs, err := packages.Load(&packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedImports | packages.NeedSyntax,
	}, path)
	if err != nil {
		return nil, fmt.Errorf("parse: could not import package %v: %w", path, err)
	}
	if len(pkgs) == 0 || len(pkgs[0].Syntax) == 0 {
		return nil, fmt.Errorf("parse: no packages found for import %v", path)
	}
	pkg := pkgs[0]
	d.importCache[path] = pkg
	return pkg, nil
}
