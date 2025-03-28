// This is free and unencumbered software released into the public
// domain.  For more information, see <http://unlicense.org> or the
// accompanying UNLICENSE file.

package mock

import (
	"context"
	"fmt"
	"go/ast"
	"go/token"
	"strconv"
	"unicode"
)

const (
	inputFmt     = "arg%d"
	outputFmt    = "ret%d"
	receiverName = "m"
	methodField  = "method"
)

func methodType() *ast.SelectorExpr {
	return &ast.SelectorExpr{
		X: &ast.Ident{
			Name: "vegr",
		},
		Sel: &ast.Ident{
			Name: "Method",
		},
	}
}

func methodInit() *ast.SelectorExpr {
	return &ast.SelectorExpr{
		X: &ast.Ident{
			Name: "vegr",
		},
		Sel: &ast.Ident{
			Name: "NewMethod",
		},
	}
}

func panicField() *ast.Field {
	return &ast.Field{
		Names: []*ast.Ident{{Name: "Panic_"}},
		Type:  &ast.Ident{Name: "any"},
	}
}

// Method represents a method that is being mocked.
type Method struct {
	receiver   Mock
	name       string
	implements *ast.FuncType
}

// MethodFor returns a Method representing typ, using receiver as
// the Method's receiver type and name as the method name.
func MethodFor(ctx context.Context, receiver Mock, name string, typ *ast.FuncType) Method {
	return Method{
		receiver:   receiver,
		name:       name,
		implements: typ,
	}
}

// Ast returns the ast representation of m.
func (m Method) Ast(ctx context.Context) *ast.FuncDecl {
	f := &ast.FuncDecl{}
	f.Name = &ast.Ident{Name: m.name}
	f.Type = m.mockType(ctx)
	f.Recv = m.recv(ctx)
	f.Body = m.body(ctx)
	return f
}

func (m Method) inName(ctx context.Context) string {
	return fmt.Sprintf("%s_%s_In", m.receiver.Name(ctx), m.name)
}

func (m Method) inType(ctx context.Context) ast.Expr {
	if len(m.receiver.typeParams) == 0 {
		return &ast.Ident{Name: m.inName(ctx)}
	}
	typ := &ast.IndexListExpr{
		X: &ast.Ident{Name: m.inName(ctx)},
	}
	for _, t := range m.receiver.typeParams {
		for _, n := range t.Names {
			typ.Indices = append(typ.Indices, n)
		}
	}
	return typ
}

func (m Method) outName(ctx context.Context) string {
	return fmt.Sprintf("%s_%s_Out", m.receiver.Name(ctx), m.name)
}

func (m Method) outType(ctx context.Context) ast.Expr {
	if len(m.receiver.typeParams) == 0 {
		return &ast.Ident{Name: m.outName(ctx)}
	}
	typ := &ast.IndexListExpr{
		X: &ast.Ident{Name: m.outName(ctx)},
	}
	for _, t := range m.receiver.typeParams {
		for _, n := range t.Names {
			typ.Indices = append(typ.Indices, n)
		}
	}
	return typ
}

func (m Method) aliasName(ctx context.Context) string {
	return fmt.Sprintf("%s_%s", m.receiver.Name(ctx), m.name)
}

func (m Method) aliasType(ctx context.Context) ast.Expr {
	if len(m.receiver.typeParams) == 0 {
		return &ast.Ident{Name: m.aliasName(ctx)}
	}
	typ := &ast.IndexListExpr{
		X: &ast.Ident{Name: m.aliasName(ctx)},
	}
	for _, t := range m.receiver.typeParams {
		for _, n := range t.Names {
			typ.Indices = append(typ.Indices, n)
		}
	}
	return typ
}

// Field returns the field that needs to be a part of the method struct for
// recording calls to this method.
func (m Method) Field(ctx context.Context) *ast.Field {
	return &ast.Field{
		Names: []*ast.Ident{{Name: m.name}},
		Type:  m.aliasType(ctx),
	}
}

func (m Method) FieldInit(ctx context.Context, buffer int) *ast.AssignStmt {
	args := []ast.Expr{
		&ast.BasicLit{Kind: token.STRING, Value: strconv.Quote(m.receiver.Name(ctx))},
		&ast.BasicLit{Kind: token.STRING, Value: strconv.Quote(m.name)},
		&ast.BasicLit{Kind: token.INT, Value: strconv.Itoa(buffer)},
	}

	params := m.implements.Params.List
	if len(params) > 0 {
		if _, ok := params[len(params)-1].Type.(*ast.Ellipsis); ok {
			args = append(args, &ast.CallExpr{
				Fun: &ast.SelectorExpr{X: &ast.Ident{Name: "vegr"}, Sel: &ast.Ident{Name: "Variadic"}},
			})
		}
	}
	return &ast.AssignStmt{
		Lhs: []ast.Expr{selectors(ctx, receiverName, methodField, m.name, "Method")},
		Tok: token.ASSIGN,
		Rhs: []ast.Expr{&ast.CallExpr{
			Fun: &ast.IndexListExpr{
				X: methodInit(),
				Indices: []ast.Expr{
					m.inType(ctx),
					m.outType(ctx),
				},
			},
			Args: args,
		}},
	}
}

// In returns the input struct for reporting method calls to m.
func (m Method) In(ctx context.Context) *ast.TypeSpec {
	params := m.params(ctx, false)
	var fields []*ast.Field
	for _, p := range params {
		f := &ast.Field{
			Type: p.Type,
		}
		if el, ok := p.Type.(*ast.Ellipsis); ok {
			f.Type = &ast.ArrayType{
				Elt: el.Elt,
			}
		}
		fields = append(fields, f)
		for _, n := range p.Names {
			nameRunes := []rune(n.Name)
			exported := string(append([]rune{unicode.ToTitle(nameRunes[0])}, nameRunes[1:]...))
			f.Names = append(f.Names, &ast.Ident{Name: exported})
		}
	}
	spec := &ast.TypeSpec{
		Name: &ast.Ident{Name: m.inName(ctx)},
		Type: &ast.StructType{Fields: &ast.FieldList{List: fields}},
	}
	if len(m.receiver.typeParams) == 0 {
		return spec
	}
	spec.TypeParams = &ast.FieldList{List: m.receiver.typeParams}
	return spec
}

// Out returns the output struct for enqueueing method returns to m.
func (m Method) Out(ctx context.Context) *ast.TypeSpec {
	results := m.results(ctx, false)
	fields := []*ast.Field{
		panicField(),
	}
	for _, p := range results {
		f := &ast.Field{
			Type: p.Type,
		}
		fields = append(fields, f)
		for _, n := range p.Names {
			nameRunes := []rune(n.Name)
			exported := string(append([]rune{unicode.ToTitle(nameRunes[0])}, nameRunes[1:]...))
			f.Names = append(f.Names, &ast.Ident{Name: exported})
		}
	}
	spec := &ast.TypeSpec{
		Name: &ast.Ident{Name: m.outName(ctx)},
		Type: &ast.StructType{Fields: &ast.FieldList{List: fields}},
	}
	if len(m.receiver.typeParams) == 0 {
		return spec
	}
	spec.TypeParams = &ast.FieldList{List: m.receiver.typeParams}
	return spec
}

// Wrapper returns the spec to wrap vegr.Method[inType, outType] as a local type.
func (m Method) Wrapper(ctx context.Context) *ast.TypeSpec {
	spec := &ast.TypeSpec{
		Name: &ast.Ident{Name: m.aliasName(ctx)},
		Type: &ast.StructType{
			Fields: &ast.FieldList{List: []*ast.Field{{
				Type: &ast.IndexListExpr{
					X: methodType(),
					Indices: []ast.Expr{
						m.inType(ctx),
						m.outType(ctx),
					},
				},
			}}},
		},
	}
	if len(m.receiver.typeParams) == 0 {
		return spec
	}
	spec.TypeParams = &ast.FieldList{List: m.receiver.typeParams}
	return spec
}

func (m Method) recv(ctx context.Context) *ast.FieldList {
	return &ast.FieldList{
		List: []*ast.Field{
			{
				Names: []*ast.Ident{{Name: receiverName}},
				Type: &ast.StarExpr{
					X: m.receiver.typeExpr(ctx),
				},
			},
		},
	}
}

func (m Method) mockType(ctx context.Context) *ast.FuncType {
	newTyp := &ast.FuncType{}
	if m.implements.Params != nil {
		newTyp.Params = &ast.FieldList{
			List: m.params(ctx, true),
		}
	}
	if m.implements.Results != nil {
		newTyp.Results = &ast.FieldList{
			List: m.results(ctx, true),
		}
	}
	return newTyp
}

func mockField(ctx context.Context, idx int, f *ast.Field, fieldFmt string, avoidCollision bool) *ast.Field {
	if len(f.Names) == 1 && f.Names[0].Name == "_" {
		f.Names = nil
	}
	if len(f.Names) == 0 {
		if idx < 0 {
			return f
		}
		// Edit the field directly to ensure the same name is used in the mock
		// struct.
		f.Names = []*ast.Ident{{Name: fmt.Sprintf(fieldFmt, idx)}}
		return f
	}

	if !avoidCollision {
		return f
	}

	// Here, we want a copy, so that we can use altered names without affecting
	// field names in the mock struct.
	newField := &ast.Field{Type: f.Type}
	for _, n := range f.Names {
		name := n.Name
		if name == receiverName {
			name += "_"
		}
		newField.Names = append(newField.Names, &ast.Ident{Name: name})
	}
	return newField
}

func (m Method) params(ctx context.Context, avoidCollision bool) []*ast.Field {
	var params []*ast.Field
	for idx, f := range m.implements.Params.List {
		params = append(params, mockField(ctx, idx, f, inputFmt, avoidCollision))
	}
	return params
}

func (m Method) results(ctx context.Context, avoidColl bool) []*ast.Field {
	if m.implements.Results == nil {
		return nil
	}
	var fields []*ast.Field
	for idx, f := range m.implements.Results.List {
		fields = append(fields, mockField(ctx, idx, f, outputFmt, avoidColl))
	}
	return fields
}

func (m Method) input(ctx context.Context) ast.Stmt {
	var elts []ast.Expr
	for _, param := range m.params(ctx, true) {
		for _, n := range param.Names {
			// Undo our hack to avoid name collisions with the receiver.
			name := n.Name
			if name == receiverName+"_" {
				name = receiverName
			}
			runes := []rune(name)
			runes[0] = unicode.ToTitle(runes[0])
			elts = append(elts, &ast.KeyValueExpr{
				Key:   &ast.Ident{Name: string(runes)},
				Value: n,
			})
		}
	}

	return &ast.SendStmt{
		Chan: &ast.CallExpr{
			Fun: selectors(ctx, receiverName, methodField, m.name, "In"),
		},
		Value: &ast.CompositeLit{
			Type: m.inType(ctx),
			Elts: elts,
		},
	}
}

// PrependLocalPackage prepends name as the package name for local types
// in m's signature.  This is most often used when mocking types that are
// imported by the local package.
func (m Method) PrependLocalPackage(ctx context.Context, name string) error {
	if err := m.prependPackage(ctx, name, m.implements.Results); err != nil {
		return fmt.Errorf("error adding package selector to results: %w", err)
	}
	if err := m.prependPackage(ctx, name, m.implements.Params); err != nil {
		return fmt.Errorf("error adding package selector to params: %w", err)
	}
	return nil
}

func (m Method) prependPackage(ctx context.Context, name string, fields *ast.FieldList) error {
	if fields == nil {
		return nil
	}
	for _, field := range fields.List {
		typ, err := m.prependTypePackage(ctx, name, field.Type)
		if err != nil {
			return fmt.Errorf("could not add package selector to type %v: %w", field.Type, err)
		}
		field.Type = typ
	}
	return nil
}

func (m Method) prependTypePackage(ctx context.Context, name string, typ ast.Expr) (ast.Expr, error) {
	switch src := typ.(type) {
	case *ast.Ident:
		if !unicode.IsUpper(rune(src.String()[0])) {
			switch src.String() {
			case "bool",
				"string",
				"int", "int8", "int16", "int32", "int64",
				"uint", "uint8", "uint16", "uint32", "uint64", "uintptr",
				"byte",
				"rune",
				"float32", "float64",
				"complex64", "complex128",
				"error", "any":
				return src, nil
			default:
				return nil, fmt.Errorf("cannot add package selector to type %v: %w", typ, ErrUnexported)
			}
		}
		for _, p := range m.receiver.typeParams {
			for _, n := range p.Names {
				if n.Name == src.Name {
					// This is a generic type name and should not have the package name prefixed.
					return src, nil
				}
			}
		}
		return selectors(ctx, name, src.String()), nil
	case *ast.FuncType:
		if err := m.prependPackage(ctx, name, src.Params); err != nil {
			return nil, fmt.Errorf("cannot add package selector to function params: %w", err)
		}
		if err := m.prependPackage(ctx, name, src.Results); err != nil {
			return nil, fmt.Errorf("cannot add package selector to function results: %w", err)
		}
		return src, nil
	case *ast.ArrayType:
		elt, err := m.prependTypePackage(ctx, name, src.Elt)
		if err != nil {
			return nil, err
		}
		src.Elt = elt
		return src, nil
	case *ast.MapType:
		key, err := m.prependTypePackage(ctx, name, src.Key)
		if err != nil {
			return nil, err
		}
		src.Key = key
		val, err := m.prependTypePackage(ctx, name, src.Value)
		if err != nil {
			return nil, err
		}
		src.Value = val
		return src, nil
	case *ast.StarExpr:
		x, err := m.prependTypePackage(ctx, name, src.X)
		if err != nil {
			return nil, err
		}
		src.X = x
		return src, nil
	case *ast.Ellipsis:
		elt, err := m.prependTypePackage(ctx, name, src.Elt)
		if err != nil {
			return nil, err
		}
		src.Elt = elt
		return src, nil
	case *ast.ChanType:
		val, err := m.prependTypePackage(ctx, name, src.Value)
		if err != nil {
			return nil, err
		}
		src.Value = val
		return src, nil
	default:
		return typ, nil
	}
}

func (m Method) returns(ctx context.Context) (stmts []ast.Stmt) {
	populateArgs := []ast.Expr{
		&ast.SelectorExpr{
			X: &ast.Ident{
				Name: receiverName,
			},
			Sel: &ast.Ident{
				Name: "t",
			},
		},
		selectors(ctx, receiverName, "timeout"),
		selectors(ctx, receiverName, methodField, m.name),
	}
	populate := &ast.CallExpr{
		Fun:  selectors(ctx, "vegr", "PopulateReturns"),
		Args: populateArgs,
	}
	if m.implements.Results == nil {
		return []ast.Stmt{&ast.ExprStmt{X: populate}}
	}

	names := m.results(ctx, true)
	for _, ns := range names {
		for _, n := range ns.Names {
			populateArgs = append(populateArgs, &ast.UnaryExpr{
				Op: token.AND,
				X:  n,
			})
		}
	}
	populate.Args = populateArgs
	ret := &ast.ReturnStmt{}
	for _, ns := range names {
		for _, n := range ns.Names {
			ret.Results = append(ret.Results, n)
		}
	}
	return []ast.Stmt{
		&ast.ExprStmt{X: populate},
		ret,
	}
}

func (m Method) body(ctx context.Context) *ast.BlockStmt {
	stmts := []ast.Stmt{
		&ast.ExprStmt{
			X: &ast.CallExpr{
				Fun: selectors(ctx, receiverName, "t", "Helper"),
			},
		},
	}
	stmts = append(stmts, m.input(ctx))
	if returnStmts := m.returns(ctx); returnStmts != nil {
		stmts = append(stmts, m.returns(ctx)...)
	}
	return &ast.BlockStmt{
		List: stmts,
	}
}
