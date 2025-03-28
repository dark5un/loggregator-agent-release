// This is free and unencumbered software released into the public
// domain.  For more information, see <http://unlicense.org> or the
// accompanying UNLICENSE file.

package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"git.sr.ht/~nelsam/hel/console"
	"git.sr.ht/~nelsam/hel/mock"
	"git.sr.ht/~nelsam/hel/parse"
	"git.sr.ht/~nelsam/hel/typ"
	"github.com/spf13/cobra"
)

var (
	cmd = &cobra.Command{
		Use:   "hel",
		Short: "A mock generator for Go",
		Long:  "The Go mock generator that you don't have to think about",
		RunE:  runE,
	}
	output     string
	chanSize   int
	modes      []string
	nonTestPkg bool
	tgtTypes   []string
	verbose    bool

	usageErr = errors.New("usage error")
)

func init() {
	cmd.Flags().StringVarP(&output, "output", "o", "mock_helheim_test.go", "The file to write generated mocks to. Since hel does "+
		"not generate exported types, this file will be saved directly in all packages with generated mocks.")
	cmd.Flags().IntVarP(&chanSize, "chan-size", "s", 100, "The size of channels used for method calls.")
	cmd.Flags().StringSliceVarP(&modes, "mode", "m", []string{"deps"}, "The mode used to detect interface types to mock. Valid values: 'deps' to search for "+
		"interfaces in the dependency tree; 'local' to search for interface types in the local code. Defaults to 'deps'.")
	cmd.Flags().BoolVar(&nonTestPkg, "no-test-package", false, "Generate mocks in the primary package rather than in {pkg}_test.")
	cmd.Flags().StringSliceVarP(&tgtTypes, "type", "t", []string{}, "The type(s) to generate mocks for. Implies --mode=local.")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output about each mock being generated, the interface type it intends to mock, and the reason it was chosen.")
}

func makeMocks(ctx context.Context, types *typ.Dir, fileName string, chanSize int, useTestPkg bool, opts ...mock.Opt) (filePath string, err error) {
	mocks, err := mock.Generate(ctx, types, opts...)
	if err != nil {
		return "", fmt.Errorf("failed to generate mocks: %w", err)
	}
	if mocks.Len(ctx) == 0 {
		return "", nil
	}
	if useTestPkg {
		mocks.PrependLocalPackage(ctx, types.Package(ctx))
	}
	filePath = filepath.Join(types.Dir(ctx), fileName)
	f, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to create output file: %w", err)
	}
	defer f.Close()
	testPkg := types.Package(ctx)
	if useTestPkg {
		testPkg += "_test"
	}
	if err := mocks.Output(ctx, testPkg, types.Dir(ctx), chanSize, f); err != nil {
		return "", fmt.Errorf("failed to write mocks to output: %w", err)
	}
	return filePath, nil
}

func progress[T any](verbose bool, f func() T) T {
	if verbose {
		// We don't want to clutter verbose output with dot characters.
		fmt.Println()
		return f()
	}

	stop := make(chan struct{})
	done := showProgress(stop)
	defer func() {
		close(stop)
		<-done
	}()

	return f()
}

func showProgress(stop <-chan struct{}) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(time.Second / 2)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				fmt.Print(".")
			case <-stop:
				return
			}
		}
	}()
	return done
}

type lengther interface {
	Len() int
}

func pluralize(values any, singular, plural string) string {
	length := findLength(values)
	if length == 1 {
		return singular
	}
	return plural
}

func findLength(values any) int {
	if lengther, ok := values.(lengther); ok {
		return lengther.Len()
	}
	return reflect.ValueOf(values).Len()
}

// runE wraps run. If run returns an error that does not wrap usageErr, then
// runE will print the error and exit. It only returns the error if it wraps
// usageErr.
func runE(cmd *cobra.Command, pkgs []string) error {
	err := run(cmd, pkgs)
	if errors.Is(err, usageErr) {
		return err
	}
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	return nil
}

func run(cmd *cobra.Command, pkgs []string) error {
	// TODO: lazy-import instead of goimports
	fmtPath, err := exec.Command("which", "goimports").Output()
	if err != nil {
		return fmt.Errorf("Command 'goimports' not found in $PATH: %w", err)
	}

	goimportsPath := strings.TrimSpace(string(fmtPath))

	defer fmt.Print("\n\n")

	if len(tgtTypes) > 0 {
		if len(pkgs) > 1 {
			return fmt.Errorf("%w: cannot specify types to mock in multiple packages; got types %v, packages %v", usageErr, tgtTypes, pkgs)
		}
		modes = []string{"local"}
	}
	var opts []console.Opt
	if verbose {
		opts = append(opts, console.Verbose())
	}
	cons := console.NewLogger(os.Stdout, opts...)
	ctx := cons.Ctx(context.Background())

	fmt.Printf("Parsing directories matching %s %v", pluralize(pkgs, "pattern", "patterns"), pkgs)
	pkgOut := progress(verbose, func() (out struct {
		err  error
		dirs []parse.Dir
	}) {
		pkgs, err := parse.Packages(ctx, pkgs...)
		if err != nil {
			out.err = fmt.Errorf("parsing packages failed: %w", err)
			return out
		}
		out.dirs = pkgs
		return out
	})
	if pkgOut.err != nil {
		return pkgOut.err
	}
	dirList := pkgOut.dirs
	fmt.Print("\n")
	fmt.Println("Found directories:")
	for _, dir := range dirList {
		fmt.Println("  " + dir.Path(ctx))
	}
	fmt.Print("\n")

	fmt.Printf("Loading interface types in matching directories")
	typeDirs := progress(verbose, func() typ.Dirs {
		var godirs []typ.GoDir
		for _, dir := range dirList {
			godirs = append(godirs, dir)
		}
		return typ.Load(ctx, nonTestPkg, godirs...).Filter(ctx, tgtTypes...)
	})
	fmt.Print("\n\n")

	fmt.Printf("Generating mocks")
	err = progress(verbose, func() error {
		var opts []mock.Opt
		for _, m := range modes {
			switch m {
			case "local":
				opts = append(opts, mock.ForLocalInterfaces())
			case "deps":
				opts = append(opts, mock.ForConcreteDependencies())
			default:
				return fmt.Errorf("%w: unsupported mode: %q", usageErr, m)
			}
		}
		for _, typeDir := range typeDirs.Slice(ctx) {
			mockPath, err := makeMocks(ctx, typeDir, output, chanSize, !nonTestPkg, opts...)
			if err != nil {
				return err
			}
			if mockPath == "" {
				continue
			}
			if err := exec.Command(goimportsPath, "-w", mockPath).Run(); err != nil {
				return fmt.Errorf("%q: goimports failed: %w", mockPath, err)
			}
			fmt.Printf("\n  Created %q", mockPath)
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func main() {
	cmd.Execute()
}
