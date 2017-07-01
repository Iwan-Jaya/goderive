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

// Package copyto contains the implementation of the copyto plugin, which generates the deriveCopyTo function.
//
// The deriveCopyTo function is a maintainable and fast way to implement fast copy functions.
//
// When goderive walks over your code it is looking for a function that:
//  - was not implemented (or was previously derived) and
//  - has a prefix predefined prefix.
//
// In the following code the deriveCopyTo function will be found, because
// it was not implemented and it has a prefix deriveCopyTo.
// This prefix is configurable.
//
//	package main
//
//	import "sort"
//
//	type MyStruct struct {
// 		Int64     int64
//		StringPtr *string
//	}
//
//	func (m *MyStruct) Clone() *MyStruct {
//		if m == nil {
//			return nil
//		}
//		n := &MyStruct{}
//		deriveCopyTo(m, n)
//		return n
//	}
//
// The initial type that is passed into deriveCopyTo needs to have a reference type:
//	- pointer
//	- slice
//	- map
// , otherwise we are not able to modify the input parameter and then what are you really copying,
// but as we go deeper we support most types.
//
// Supported types:
//	- basic types
//	- named structs
//	- slices
//	- maps
//	- pointers to these types
//	- private fields of structs in external packages (using reflect and unsafe)
//	- and many more
// Unsupported types:
//	- chan
//	- interface
//	- function
//	- unnamed structs, which are not comparable with the == operator
//
// This plugin has been tested thoroughly.
package copyto

import (
	"fmt"
	"go/types"
	"strings"

	"github.com/awalterschulze/goderive/derive"
)

// NewPlugin creates a new copyto plugin.
// This function returns the plugin name, default prefix and a constructor for the copyto code generator.
func NewPlugin() derive.Plugin {
	return derive.NewPlugin("copyto", "deriveCopyTo", New)
}

// New is a constructor for the copyto code generator.
// This generator should be reconstructed for each package.
func New(typesMap derive.TypesMap, p derive.Printer, deps map[string]derive.Dependency) derive.Generator {
	return &gen{
		TypesMap:   typesMap,
		printer:    p,
		bytesPkg:   p.NewImport("bytes"),
		reflectPkg: p.NewImport("reflect"),
		unsafePkg:  p.NewImport("unsafe"),
	}
}

type gen struct {
	derive.TypesMap
	printer    derive.Printer
	bytesPkg   derive.Import
	reflectPkg derive.Import
	unsafePkg  derive.Import
}

func (this *gen) Add(name string, typs []types.Type) (string, error) {
	if len(typs) != 2 {
		return "", fmt.Errorf("%s does not have two arguments", name)
	}
	if !types.Identical(typs[0], typs[1]) {
		return "", fmt.Errorf("%s has two arguments, but they are of different types %s != %s",
			name, this.TypeString(typs[0]), this.TypeString(typs[1]))
	}
	return this.SetFuncName(name, typs[0])
}

func (this *gen) Generate(typs []types.Type) error {
	return this.genFunc(typs[0])
}

func (g *gen) genFunc(typ types.Type) error {
	p := g.printer
	g.Generating(typ)
	typeStr := g.TypeString(typ)
	p.P("")
	p.P("func %s(this, that %s) {", g.GetFuncName(typ), typeStr)
	p.In()
	if err := g.genStatement(typ, "this", "that"); err != nil {
		return err
	}
	p.Out()
	p.P("}")
	return nil
}

func (g *gen) genStatement(typ types.Type, this, that string) error {
	p := g.printer
	if canCopy(typ) {
		p.P("%s = %s", that, this)
		return nil
	}
	switch ttyp := typ.Underlying().(type) {
	case *types.Pointer:
		reftyp := ttyp.Elem()
		g.TypeString(reftyp)
		thisref, thatref := "*"+this, "*"+that
		_, isNamed := reftyp.(*types.Named)
		strct, isStruct := reftyp.Underlying().(*types.Struct)
		if !isStruct {
			if err := g.genField(reftyp, thisref, thatref); err != nil {
				return err
			}
			return nil
		} else if isNamed {
			fields := derive.Fields(g.TypesMap, strct)
			if len(fields.Fields) > 0 {
				thisv := prepend(this, "v")
				thatv := prepend(that, "v")
				if fields.Reflect {
					p.P(thisv+` := `+g.reflectPkg()+`.Indirect(`+g.reflectPkg()+`.ValueOf(%s))`, this)
					p.P(thatv+` := `+g.reflectPkg()+`.Indirect(`+g.reflectPkg()+`.ValueOf(%s))`, that)
				}
				for _, field := range fields.Fields {
					fieldType := field.Type
					var thisField, thatField string
					if !field.Private() {
						thisField, thatField = field.Name(this, nil), field.Name(that, nil)
					} else {
						thisField, thatField = field.Name(thisv, g.unsafePkg), field.Name(thatv, g.unsafePkg)
					}
					if err := g.genField(fieldType, thisField, thatField); err != nil {
						return err
					}
				}
			}
			return nil
		}
	case *types.Slice:
		elmType := ttyp.Elem()
		if canCopy(elmType) {
			p.P("copy(%s, %s)", that, this)
			return nil
		}
		thisvalue := prepend(this, "value")
		thisi := prepend(this, "i")
		p.P("for %s, %s := range %s {", thisi, thisvalue, this)
		p.In()
		if err := g.genField(elmType, thisvalue, wrap(that)+"["+thisi+"]"); err != nil {
			return err
		}
		p.Out()
		p.P("}")
		return nil
	case *types.Array:
		elmType := ttyp.Elem()
		thisvalue := prepend(this, "value")
		thisi := prepend(this, "i")
		p.P("for %s, %s := range %s {", thisi, thisvalue, this)
		p.In()
		if err := g.genField(elmType, thisvalue, wrap(that)+"["+thisi+"]"); err != nil {
			return err
		}
		p.Out()
		p.P("}")
		return nil
	case *types.Map:
		elmType := ttyp.Elem()
		keyType := ttyp.Key()
		thiskey, thisvalue := prepend(this, "key"), prepend(this, "value")
		p.P("for %s, %s := range %s {", thiskey, thisvalue, this)
		p.In()
		thatkey := thiskey
		if !canCopy(keyType) {
			if err := g.genField(keyType, thatkey, thiskey); err != nil {
				return err
			}
			thatkey = prepend(that, "key")
		}
		if nullable(elmType) {
			p.P("if %s == nil {", thisvalue)
			p.In()
			p.P("%s = nil", wrap(that)+"["+thatkey+"]")
			p.Out()
			p.P("}")
		}
		if err := g.genField(elmType, thisvalue, wrap(that)+"["+thatkey+"]"); err != nil {
			return err
		}
		p.Out()
		p.P("}")
		return nil
	}
	return fmt.Errorf("unsupported copyto type: %s", g.TypeString(typ))
}

func nullable(typ types.Type) bool {
	switch typ.(type) {
	case *types.Pointer, *types.Slice, *types.Map:
		return true
	}
	return false
}

func not(s string) string {
	if strings.HasPrefix(s, "(") && strings.HasSuffix(s, ")") {
		return "!" + s
	}
	return "!(" + s + ")"
}

func wrap(value string) string {
	if strings.HasPrefix(value, "*") ||
		strings.HasPrefix(value, "&") ||
		strings.HasSuffix(value, "]") {
		return "(" + value + ")"
	}
	return value
}

func prepend(before, after string) string {
	bs := strings.Split(before, ".")
	b := strings.Replace(bs[0], "*", "", -1)
	return b + "_" + after
}

func canCopy(tt types.Type) bool {
	t := tt.Underlying()
	switch typ := t.(type) {
	case *types.Basic:
		return typ.Kind() != types.UntypedNil
	case *types.Struct:
		for i := 0; i < typ.NumFields(); i++ {
			f := typ.Field(i)
			ft := f.Type()
			if !canCopy(ft) {
				return false
			}
		}
		return true
	case *types.Array:
		return canCopy(typ.Elem())
	}
	return false
}

func hasCopyToMethod(typ *types.Named) bool {
	for i := 0; i < typ.NumMethods(); i++ {
		meth := typ.Method(i)
		if meth.Name() != "CopyTo" {
			continue
		}
		sig, ok := meth.Type().(*types.Signature)
		if !ok {
			// impossible, but lets check anyway
			continue
		}
		if sig.Params().Len() != 1 {
			continue
		}
		res := sig.Results()
		if res.Len() != 0 {
			continue
		}
		return true
	}
	return false
}

func (g *gen) genField(fieldType types.Type, thisField, thatField string) error {
	p := g.printer
	if canCopy(fieldType) {
		p.P("%s = %s", thatField, thisField)
		return nil
	}
	switch typ := fieldType.Underlying().(type) {
	case *types.Pointer:
		p.P("if %s == nil {", thisField)
		p.In()
		p.P("%s = nil", thatField)
		p.Out()
		p.P("} else {")
		p.In()
		ref := typ.Elem()
		p.P("%s = new(%s)", thatField, g.TypeString(typ.Elem()))
		if named, ok := ref.(*types.Named); ok && hasCopyToMethod(named) {
			p.P("%s.CopyTo(%s)", wrap(thisField), thatField)
		} else if canCopy(typ.Elem()) {
			p.P("*%s = *%s", thatField, thisField)
		} else {
			p.P("%s(%s, %s)", g.GetFuncName(typ), thisField, thatField)
		}
		p.Out()
		p.P("}")
		return nil
	case *types.Array:
		g.genStatement(fieldType, thisField, thatField)
		return nil
	case *types.Slice:
		p.P("if %s == nil {", thisField) // nil
		p.In()
		p.P("%s = nil", thatField)
		p.Out()
		p.P("} else {") // nil
		p.In()
		p.P("if %s != nil {", thatField) // not nil
		p.In()
		p.P("if len(%s) > len(%s) {", thisField, thatField) // len
		p.In()
		p.P("if cap(%s) >= len(%s) {", thatField, thisField) // cap
		p.In()
		p.P("%s = (%s)[:len(%s)]", thatField, thatField, thisField)
		p.Out()
		p.P("} else {") // cap
		p.In()
		p.P("%s = make(%s, len(%s))", thatField, g.TypeString(typ), thisField)
		p.Out()
		p.P("}")
		p.Out()
		p.P("} else if len(%s) < len(%s) {", thisField, thatField) // len
		p.In()
		p.P("%s = (%s)[:len(%s)]", thatField, thatField, thisField)
		p.Out()
		p.P("}") // len
		p.Out()
		p.P("} else {") // not nil
		p.In()
		p.P("%s = make(%s, len(%s))", thatField, g.TypeString(typ), thisField)
		p.Out()
		p.P("}") // not nil
		if canCopy(typ.Elem()) {
			p.P("copy(%s, %s)", thatField, thisField)
		} else {
			p.P("%s(%s, %s)", g.GetFuncName(typ), thisField, thatField)
		}
		p.Out()
		p.P("}") // nil
		return nil
	case *types.Map:
		p.P("if %s != nil {", thisField)
		p.In()
		p.P("%s = make(%s, len(%s))", thatField, g.TypeString(typ), thisField)
		p.P("%s(%s, %s)", g.GetFuncName(typ), thisField, thatField)
		p.Out()
		p.P("} else {")
		p.In()
		p.P("%s = nil", thatField)
		p.Out()
		p.P("}")
		return nil
	case *types.Struct:
		p.P("field := new(%s)", g.TypeString(fieldType))
		named, isNamed := fieldType.(*types.Named)
		if isNamed && hasCopyToMethod(named) {
			p.P("%s.CopyTo(field)", wrap(thisField))
		} else {
			p.P("%s(%s, %s)", g.GetFuncName(fieldType), thisField, thatField)
		}
		p.P("%s = *field", thatField)
		return nil
	default: // *Chan, *Tuple, *Signature, *Interface, *types.Basic.Kind() == types.UntypedNil, *Struct
		return fmt.Errorf("unsupported field type %s", g.TypeString(fieldType))
	}
}
