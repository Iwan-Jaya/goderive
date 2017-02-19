//  Copyright 2017 Walter Schulze
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.

package main

import (
	"fmt"
	"go/ast"
	"go/types"
	"strings"

	"golang.org/x/tools/go/loader"
)

const eqFuncPrefix = "deriveEqual"

func generateEqual(p Printer, pkgInfo *loader.PackageInfo, calls []*ast.CallExpr) error {
	qual := types.RelativeTo(pkgInfo.Pkg)
	m := newTypesMap(qual)

	for _, call := range calls {
		fn, ok := call.Fun.(*ast.Ident)
		if !ok {
			continue
		}
		if !strings.HasPrefix(fn.Name, eqFuncPrefix) {
			continue
		}
		if len(call.Args) != 2 {
			return fmt.Errorf("%s does not have two arguments\n", fn.Name)
		}
		t0 := pkgInfo.TypeOf(call.Args[0])
		t1 := pkgInfo.TypeOf(call.Args[1])
		if !types.Identical(t0, t1) {
			return fmt.Errorf("%s has two arguments, but they are of different types %s != %s\n",
				fn.Name, t0, t1)
		}
		name := strings.TrimPrefix(fn.Name, eqFuncPrefix)
		qual := types.RelativeTo(pkgInfo.Pkg)
		typeStr := typeName(t0, qual)
		if typeStr != name {
			//TODO think about whether this is really necessary
			return fmt.Errorf("%s's suffix %s does not match the type %s\n",
				fn.Name, name, typeStr)
		}
		m.Set(t0, false)
	}

	eq := newEqual(p, m, qual, eqFuncPrefix)

	for _, typ := range m.List() {
		if err := eq.genFuncFor(typ); err != nil {
			return err
		}
	}
	for _, typ := range m.List() {
		if m.Get(typ) {
			continue
		}
		if err := eq.genFuncFor(typ); err != nil {
			return err
		}
	}
	return nil
}

func newEqual(printer Printer, typesMap TypesMap, qual types.Qualifier, prefix string) *equal {
	return &equal{
		printer:  printer,
		typesMap: typesMap,
		qual:     qual,
		bytesPkg: printer.NewImport("bytes"),
		prefix:   prefix,
	}
}

type equal struct {
	printer  Printer
	typesMap TypesMap
	qual     types.Qualifier
	bytesPkg Import
	prefix   string
}

func (this *equal) funcName(typ types.Type) string {
	return eqFuncPrefix + typeName(typ, this.qual)
}

func (this *equal) genFuncFor(typ types.Type) error {
	p := this.printer
	m := this.typesMap
	m.Set(typ, true)
	typeStr := types.TypeString(typ, this.qual)
	p.P("")
	p.P("func %s(this, that %s) bool {", this.funcName(typ), typeStr)
	p.In()
	switch ttyp := typ.(type) {
	case *types.Pointer:
		ref := ttyp.Elem()
		switch tttyp := ref.Underlying().(type) {
		case *types.Basic:
			fieldStr, err := this.field("this", "that", typ)
			if err != nil {
				return err
			}
			p.P("return " + fieldStr)
		case *types.Slice, *types.Array, *types.Map:
			if !this.typesMap.Get(tttyp) {
				this.typesMap.Set(tttyp, false)
			}
			p.P("return (this == nil && that == nil) || (this != nil) && (that != nil) && %s(%s, %s)", this.funcName(tttyp), "*this", "*that")
		case *types.Struct:
			numFields := tttyp.NumFields()
			if numFields == 0 {
				p.P("return (this == nil && that == nil) || (this != nil) && (that != nil)")
			} else {
				p.P("return (this == nil && that == nil) || (this != nil) && (that != nil) &&")
			}
			p.In()
			for i := 0; i < numFields; i++ {
				field := tttyp.Field(i)
				fieldType := field.Type()
				fieldName := field.Name()
				thisField := "this." + fieldName
				thatField := "that." + fieldName
				fieldStr, err := this.field(thisField, thatField, fieldType)
				if err != nil {
					return err
				}
				if (i + 1) != numFields {
					p.P(fieldStr + " &&")
				} else {
					p.P(fieldStr)
				}
			}
			p.Out()
		default:
			return fmt.Errorf("unsupported: pointer is not a named struct, but %#v\n", ref)
		}
	case *types.Slice:
		p.P("if this == nil || that == nil {")
		p.In()
		p.P("return this == nil && that == nil")
		p.Out()
		p.P("}")
		p.P("if len(this) != len(that) {")
		p.In()
		p.P("return false")
		p.Out()
		p.P("}")
		p.P("for i := 0; i < len(this); i++ {")
		p.In()
		eqStr, err := this.field("this[i]", "that[i]", ttyp.Elem())
		if err != nil {
			return err
		}
		p.P("if %s {", not(eqStr))
		p.In()
		p.P("return false")
		p.Out()
		p.P("}")
		p.Out()
		p.P("}")
		p.P("return true")
	case *types.Array:
		p.P("for i := 0; i < len(this); i++ {")
		p.In()
		eqStr, err := this.field("this[i]", "that[i]", ttyp.Elem())
		if err != nil {
			return err
		}
		p.P("if %s {", not(eqStr))
		p.In()
		p.P("return false")
		p.Out()
		p.P("}")
		p.Out()
		p.P("}")
		p.P("return true")
	case *types.Map:
		p.P("if this == nil || that == nil {")
		p.In()
		p.P("return this == nil && that == nil")
		p.Out()
		p.P("}")
		p.P("if len(this) != len(that) {")
		p.In()
		p.P("return false")
		p.Out()
		p.P("}")
		p.P("for k, v := range this {")
		p.In()
		p.P("thatv, ok := that[k]")
		p.P("if !ok {")
		p.In()
		p.P("return false")
		p.Out()
		p.P("}")
		eqStr, err := this.field("v", "thatv", ttyp.Elem())
		if err != nil {
			return err
		}
		p.P("if %s {", not(eqStr))
		p.In()
		p.P("return false")
		p.Out()
		p.P("}")
		p.Out()
		p.P("}")
		p.P("return true")
	default:
		return fmt.Errorf("unsupported type: %#v", typ)
	}
	p.Out()
	p.P("}")
	return nil
}

func not(s string) string {
	if s[0] == '(' {
		return "!" + s
	}
	return "!(" + s + ")"
}

func isComparable(tt types.Type) bool {
	t := tt.Underlying()
	switch typ := t.(type) {
	case *types.Basic:
		return typ.Kind() != types.UntypedNil
	case *types.Struct:
		for i := 0; i < typ.NumFields(); i++ {
			f := typ.Field(i)
			ft := f.Type()
			if !isComparable(ft) {
				return false
			}
		}
		return true
	case *types.Array:
		return isComparable(typ.Elem())
	}
	return false
}

func (this *equal) field(thisField, thatField string, fieldType types.Type) (string, error) {
	if isComparable(fieldType) {
		return fmt.Sprintf("%s == %s", thisField, thatField), nil
	}
	switch typ := fieldType.(type) {
	case *types.Pointer:
		ref := typ.Elem()
		if _, ok := ref.(*types.Named); ok {
			return fmt.Sprintf("%s.Equal(%s)", thisField, thatField), nil
		}
		eqStr, err := this.field("*"+thisField, "*"+thatField, ref)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("((%[1]s == nil && %[2]s == nil) || (%[1]s != nil && %[2]s != nil && %[3]s))", thisField, thatField, eqStr), nil
	case *types.Array:
		if !this.typesMap.Get(typ) {
			this.typesMap.Set(typ, false)
		}
		return fmt.Sprintf("%s(%s, %s)", this.funcName(typ), thisField, thatField), nil
	case *types.Slice:
		if b, ok := typ.Elem().(*types.Basic); ok && b.Kind() == types.Byte {
			return fmt.Sprintf("%s.Equal(%s, %s)", this.bytesPkg(), thisField, thatField), nil
		}
		if !this.typesMap.Get(typ) {
			this.typesMap.Set(typ, false)
		}
		return fmt.Sprintf("%s(%s, %s)", this.funcName(typ), thisField, thatField), nil
	case *types.Map:
		if !this.typesMap.Get(typ) {
			this.typesMap.Set(typ, false)
		}
		return fmt.Sprintf("%s(%s, %s)", this.funcName(typ), thisField, thatField), nil
	case *types.Named:
		return fmt.Sprintf("%s.Equal(&%s)", thisField, thatField), nil
	default: // *Chan, *Tuple, *Signature, *Interface, *types.Basic.Kind() == types.UntypedNil, *Struct
		return "", fmt.Errorf("unsupported type %#v", fieldType)
	}
}
