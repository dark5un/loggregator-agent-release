// This is free and unencumbered software released into the public
// domain.  For more information, see <http://unlicense.org> or the
// accompanying UNLICENSE file.

package mock

import (
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"unicode"
)

var ErrUnexported = errors.New("must be exported")

// Mock is a mock of an interface type.
type Mock struct {
	typeName   string
	typeParams []*ast.Field
	implements *ast.InterfaceType
}

// For returns a Mock representing typ.  An error will be returned
// if a mock cannot be created from typ.
func For(ctx context.Context, name string, params []*ast.Field, typ *ast.InterfaceType) (Mock, error) {
	return Mock{
		typeName:   name,
		implements: typ,
		typeParams: params,
	}, nil
}

// Name returns the type name for m.
func (m Mock) Name(ctx context.Context) string {
	if len(m.typeName) == 0 {
		return ""
	}
	if !unicode.IsUpper([]rune(m.typeName)[0]) {
		return "mock_" + m.typeName
	}
	return "mock" + m.typeName
}

// Methods returns the methods that need to be created with m
// as a receiver.
func (m Mock) Methods(ctx context.Context) (methods []Method) {
	for _, method := range m.implements.Methods.List {
		switch methodType := method.Type.(type) {
		case *ast.FuncType:
			methods = append(methods, MethodFor(ctx, m, method.Names[0].String(), methodType))
		}
	}
	return
}

// PrependLocalPackage prepends name as the package name for local types
// in m's signature.  This is most often used when mocking types that are
// imported by the local package.
func (m Mock) PrependLocalPackage(ctx context.Context, name string) error {
	if len(m.typeName) == 0 {
		return errors.New("cannot mock a type which does not have a name")
	}
	if !unicode.IsUpper([]rune(m.typeName)[0]) {
		return fmt.Errorf("type [%v]: %w", m.typeName, ErrUnexported)
	}
	for _, meth := range m.Methods(ctx) {
		if len(meth.name) == 0 {
			return errors.New("method has no name")
		}
		if unicode.IsLower([]rune(meth.name)[0]) {
			return fmt.Errorf("cannot add package selector [%v] to type [%v]: method [%v]: %w", name, m.Name(ctx), meth.name, ErrUnexported)
		}
		if err := meth.PrependLocalPackage(ctx, name); err != nil {
			return err
		}
	}
	return nil
}

// Constructor returns a function AST to construct m.  chanSize will be
// the buffer size for all channels initialized in the constructor.
func (m Mock) Constructor(ctx context.Context, chanSize int) *ast.FuncDecl {
	decl := &ast.FuncDecl{}
	typeRunes := []rune(m.Name(ctx))
	typeRunes[0] = unicode.ToUpper(typeRunes[0])
	decl.Name = &ast.Ident{Name: "new" + string(typeRunes)}
	typ := &ast.FuncType{
		Params: &ast.FieldList{List: []*ast.Field{
			{
				Names: []*ast.Ident{{Name: "t"}},
				Type: &ast.SelectorExpr{
					X:   &ast.Ident{Name: "vegr"},
					Sel: &ast.Ident{Name: "TestingT"},
				},
			},
			{
				Names: []*ast.Ident{{Name: "timeout"}},
				Type: &ast.SelectorExpr{
					X:   &ast.Ident{Name: "time"},
					Sel: &ast.Ident{Name: "Duration"},
				},
			},
		}},
		Results: &ast.FieldList{List: []*ast.Field{
			{
				Type: &ast.StarExpr{
					X: m.typeExpr(ctx),
				},
			},
		}},
	}
	if len(m.typeParams) > 0 {
		typ.TypeParams = &ast.FieldList{List: m.typeParams}
	}
	decl.Type = typ
	decl.Body = &ast.BlockStmt{List: m.constructorBody(ctx, chanSize)}
	return decl
}

// Decl returns the declaration AST for m.
func (m Mock) StructDecls(ctx context.Context) []ast.Decl {
	spec := &ast.TypeSpec{}
	spec.Name = &ast.Ident{Name: m.Name(ctx)}
	if len(m.typeParams) > 0 {
		spec.TypeParams = &ast.FieldList{List: m.typeParams}
	}
	spec.Type = m.structType(ctx)
	var decls []ast.Decl
	for _, meth := range m.Methods(ctx) {
		decls = append(decls,
			&ast.GenDecl{Tok: token.TYPE, Specs: []ast.Spec{meth.In(ctx)}},
			&ast.GenDecl{Tok: token.TYPE, Specs: []ast.Spec{meth.Out(ctx)}},
			&ast.GenDecl{Tok: token.TYPE, Specs: []ast.Spec{meth.Wrapper(ctx)}},
		)
	}
	return append(decls,
		&ast.GenDecl{
			Tok:   token.TYPE,
			Specs: []ast.Spec{spec},
		},
	)
}

// Ast returns all declaration AST for m.
func (m Mock) Ast(ctx context.Context, chanSize int) []ast.Decl {
	decls := append(m.StructDecls(ctx),
		m.Constructor(ctx, chanSize),
	)
	for _, method := range m.Methods(ctx) {
		decls = append(decls, method.Ast(ctx))
	}
	return decls
}

func (m Mock) constructorBody(ctx context.Context, chanSize int) []ast.Stmt {
	structAlloc := &ast.AssignStmt{
		Lhs: []ast.Expr{&ast.Ident{Name: receiverName}},
		Tok: token.DEFINE,
		Rhs: []ast.Expr{&ast.UnaryExpr{Op: token.AND, X: &ast.CompositeLit{
			Type: m.typeExpr(ctx),
			Elts: []ast.Expr{
				&ast.KeyValueExpr{
					Key:   &ast.Ident{Name: "t"},
					Value: &ast.Ident{Name: "t"},
				},
				&ast.KeyValueExpr{
					Key:   &ast.Ident{Name: "timeout"},
					Value: &ast.Ident{Name: "timeout"},
				},
			},
		}}},
	}
	stmts := []ast.Stmt{structAlloc}
	for _, method := range m.Methods(ctx) {
		stmts = append(stmts, method.FieldInit(ctx, chanSize))
	}
	stmts = append(stmts, &ast.ReturnStmt{Results: []ast.Expr{&ast.Ident{Name: receiverName}}})
	return stmts
}

func (m Mock) typeExpr(ctx context.Context) ast.Expr {
	ident := &ast.Ident{Name: m.Name(ctx)}
	if len(m.typeParams) == 0 {
		return ident
	}
	lst := &ast.IndexListExpr{X: ident}
	for _, p := range m.typeParams {
		for _, n := range p.Names {
			lst.Indices = append(lst.Indices, &ast.Ident{Name: n.Name})
		}
	}
	return lst
}

func (m Mock) structType(ctx context.Context) *ast.StructType {
	methods := &ast.StructType{
		Fields: &ast.FieldList{},
	}
	for _, method := range m.Methods(ctx) {
		methods.Fields.List = append(methods.Fields.List, method.Field(ctx))
	}
	structType := &ast.StructType{Fields: &ast.FieldList{List: []*ast.Field{
		{
			Names: []*ast.Ident{{Name: "t"}},
			Type: &ast.SelectorExpr{
				X:   &ast.Ident{Name: "vegr"},
				Sel: &ast.Ident{Name: "TestingT"},
			},
		},
		{
			Names: []*ast.Ident{{Name: "timeout"}},
			Type: &ast.SelectorExpr{
				X:   &ast.Ident{Name: "time"},
				Sel: &ast.Ident{Name: "Duration"},
			},
		},
		{
			Names: []*ast.Ident{{Name: methodField}},
			Type:  methods,
		},
	}}}
	return structType
}
