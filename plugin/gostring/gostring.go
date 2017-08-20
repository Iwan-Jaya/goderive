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

// Package gostring contains the implementation of the gostring plugin, which generates the deriveGoString function.
package gostring

import (
	"fmt"
	"go/types"

	"github.com/awalterschulze/goderive/derive"
)

// NewPlugin creates a new gostring plugin.
// This function returns the plugin name, default prefix and a constructor for the gostring code generator.
func NewPlugin() derive.Plugin {
	return derive.NewPlugin("gostring", "deriveGoString", New)
}

// New is a constructor for the gostring code generator.
// This generator should be reconstructed for each package.
func New(typesMap derive.TypesMap, p derive.Printer, deps map[string]derive.Dependency) derive.Generator {
	return &gen{
		TypesMap:   typesMap,
		printer:    p,
		strconvPkg: p.NewImport("strconv"),
		bytesPkg:   p.NewImport("bytes"),
		fmtPkg:     p.NewImport("fmt"),
	}
}

type gen struct {
	derive.TypesMap
	printer    derive.Printer
	strconvPkg derive.Import
	bytesPkg   derive.Import
	fmtPkg     derive.Import
}

func (this *gen) Add(name string, typs []types.Type) (string, error) {
	if len(typs) != 1 {
		return "", fmt.Errorf("%s does not have two arguments", name)
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
	p.P("func %s(this %s) string {", g.GetFuncName(typ), typeStr)
	p.In()
	p.P("buf := %s.NewBuffer(nil)", g.bytesPkg())
	p.P("%s.Fprintf(buf, \"func() %s {\\n\")", g.fmtPkg(), typeStr)
	if err := g.genStatement(typ, "this"); err != nil {
		return err
	}
	p.P("%s.Fprintf(buf, \"}()\\n\")", g.fmtPkg())
	p.P("return buf.String()")
	p.Out()
	p.P("}")
	return nil
}

func (g *gen) W(format string, a ...interface{}) {
	s := fmt.Sprintf(format, a...)
	g.printer.P("%s.Fprintf(buf, \"%s\\n\")", g.fmtPkg(), s)
}

func (g *gen) P(format string, a ...interface{}) {
	g.printer.P(format, a...)
}

func (g *gen) genStatement(typ types.Type, this string) error {
	p := g.printer
	switch ttyp := typ.Underlying().(type) {
	case *types.Basic:
		p.P("%s.Fprintf(buf, \"return %s\\n\", %s)", g.fmtPkg(), "%#v", this)
		return nil
	case *types.Pointer:
		p.P("if %s == nil {", this)
		p.In()
		g.W("return nil")
		p.Out()
		p.P("} else {")
		p.In()
		reftyp := ttyp.Elem()
		thisref := "*" + this
		named, isNamed := reftyp.(*types.Named)
		strct, isStruct := reftyp.Underlying().(*types.Struct)
		if !isStruct {
			g.W("%s := new(%s)", this, g.TypeString(reftyp))
			g.genField(reftyp, thisref)
			g.W("return %s", this)
		} else if isNamed {
			external := g.TypesMap.IsExternal(named)
			fields := derive.Fields(g.TypesMap, strct, external)
			if len(fields.Fields) == 0 {
				g.W("return &%s{}", g.TypeString(reftyp))
			} else {
				g.W("%s := &%s{}", this, g.TypeString(reftyp))
				for _, field := range fields.Fields {
					if field.Private() {
						return fmt.Errorf("private fields not supported, found %s in %v", field.Name("", nil), named)
					}
					thisField := field.Name(this, nil)
					if err := g.genField(field.Type, thisField); err != nil {
						return err
					}
				}
				g.W("return %s", this)
			}
		}
		p.Out()
		p.P("}")
		return nil
	case *types.Struct:

	case *types.Slice:

	case *types.Array:

	case *types.Map:

	}
	return fmt.Errorf("unsupported root type: %#v", typ)
}

func (g *gen) genField(fieldType types.Type, this string) error {
	p := g.printer
	switch typ := fieldType.Underlying().(type) {
	case *types.Basic:
		p.P("%s.Fprintf(buf, \"%s = %s\\n\", %s)", g.fmtPkg(), this, "%#v", this)
		return nil
	case *types.Pointer:
		p.P("if %s != nil {", this)
		p.In()
		p.P("%s.Fprintf(buf, \"%s = %s\\n\", %s)", g.fmtPkg(), this, "%s", g.GetFuncName(typ)+"("+this+")")
		p.Out()
		p.P("}")
		return nil
	case *types.Slice:
		p.P("if %s != nil {", this)
		p.In()
		elmTyp := typ.Elem()
		if _, isBasic := elmTyp.(*types.Basic); isBasic {
			p.P("%s.Fprintf(buf, \"%s = %s\\n\", %s)", g.fmtPkg(), this, "%#v", this)
		} else {
			p.P("%s.Fprintf(buf, \"%s = make(%s, %s)\\n\", %s)", g.fmtPkg(), this, g.TypeString(typ), "%d", "len("+this+")")
			p.P("for i := range %s {", this)
			p.In()
			goStringElm := g.GetFuncName(elmTyp)
			p.P("%s.Fprintf(buf, \"%s[%s] = %s\\n\", %s, %s)", g.fmtPkg(), this, "%d", "%s", "i", goStringElm+"("+this+"[i])")
			p.Out()
			p.P("}")
		}
		p.Out()
		p.P("}")
		return nil
	case *types.Array:
		elmTyp := typ.Elem()
		if _, isBasic := elmTyp.(*types.Basic); isBasic {
			p.P("%s.Fprintf(buf, \"%s = %s\\n\", %s)", g.fmtPkg(), this, "%#v", this)
		} else {
			p.P("for i := range %s {", this)
			p.In()
			goStringElm := g.GetFuncName(elmTyp)
			p.P("%s.Fprintf(buf, \"%s[%s] = %s\\n\", %s, %s)", g.fmtPkg(), this, "%d", "%s", "i", goStringElm+"("+this+"[i])")
			p.Out()
			p.P("}")
		}
		return nil
	}
	return fmt.Errorf("unsupported field type %#v", fieldType)
}
