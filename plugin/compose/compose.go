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

// Package compose contains the implementation of the compose plugin, which generates the deriveCompose function.
//
// The deriveCompose function composes a tuple containing an error and a function taking the value as input and returning its result, which also returns an error.
//    deriveCompose(func() (A, error), func(A) (B, error)) (B, error)
//    deriveCompose(func(A) (B, error), func(B) (C, error)) func(A) (C, error)
//    deriveCompose(func(A...) (B..., error), func(B...) (C..., error)) func(A...) (C..., error)
package compose

import (
	"fmt"
	"go/types"
	"strconv"
	"strings"

	"github.com/awalterschulze/goderive/derive"
)

// NewPlugin creates a new compose plugin.
// This function returns the plugin name, default prefix and a constructor for the compose code generator.
func NewPlugin() derive.Plugin {
	return derive.NewPlugin("compose", "deriveCompose", New)
}

// New is a constructor for the compose code generator.
// This generator should be reconstructed for each package.
func New(typesMap derive.TypesMap, p derive.Printer, deps map[string]derive.Dependency) derive.Generator {
	return &gen{
		TypesMap: typesMap,
		printer:  p,
		fmap:     deps["fmap"],
		join:     deps["join"],
	}
}

type gen struct {
	derive.TypesMap
	printer derive.Printer
	fmap    derive.Dependency
	join    derive.Dependency
}

func (g *gen) Add(name string, typs []types.Type) (string, error) {
	if len(typs) != 2 {
		return "", fmt.Errorf("%s does not have two arguments", name)
	}
	switch typs[0].(type) {
	case *types.Signature:
		_, _, _, err := g.errorType(name, typs)
		if err != nil {
			return "", err
		}
		return g.SetFuncName(name, typs...)
	}
	return "", fmt.Errorf("unsupported type %s", typs[0])
}

func (g *gen) errorType(name string, typs []types.Type) ([]types.Type, []types.Type, []types.Type, error) {
	if len(typs) != 2 {
		return nil, nil, nil, fmt.Errorf("%s does not have two arguments", name)
	}
	sig, ok := typs[0].(*types.Signature)
	if !ok {
		return nil, nil, nil, fmt.Errorf("%s, the first argument, %s, is not of type function", name, typs[0])
	}
	as := make([]types.Type, sig.Params().Len())
	for i := range as {
		as[i] = sig.Params().At(i).Type()
	}
	if sig.Results().Len() == 0 {
		return nil, nil, nil, fmt.Errorf("%s, the first function, %s, does not return any parameters", name, typs[0])
	}
	errType := sig.Results().At(sig.Results().Len() - 1).Type()
	if !derive.IsError(errType) {
		return nil, nil, nil, fmt.Errorf("%s, the first function's last result, %s, is not of type error", name, errType)
	}
	bs := make([]types.Type, sig.Results().Len()-1)
	for i := range bs {
		bs[i] = sig.Results().At(i).Type()
	}
	sig2, ok := typs[1].(*types.Signature)
	if !ok {
		return nil, nil, nil, fmt.Errorf("%s, the second argument, %s, is not of type function", name, typs[1])
	}
	if sig2.Params().Len() != len(bs) {
		return nil, nil, nil, fmt.Errorf("%s, the second function's (%s) number of input parameters do not match the first function's (%s) number of output parameters", name, typs[1], typs[0])
	}
	for i := range bs {
		b2 := sig2.Params().At(i).Type()
		if !types.AssignableTo(bs[i], b2) {
			return nil, nil, nil, fmt.Errorf("%s, the second function's (%s) input parameters types do not match the first function's (%s) output parameters types", name, typs[1], typs[0])
		}
	}
	errType2 := sig2.Results().At(sig2.Results().Len() - 1).Type()
	if !derive.IsError(errType) {
		return nil, nil, nil, fmt.Errorf("%s, the second function's last result, %s, is not of type error", name, errType2)
	}
	cs := make([]types.Type, sig2.Results().Len()-1)
	for i := range cs {
		cs[i] = sig2.Results().At(i).Type()
	}
	return as, bs, cs, nil
}

func (g *gen) Generate(typs []types.Type) error {
	switch typs[0].(type) {
	case *types.Signature:
		return g.genError(typs)
	}
	return fmt.Errorf("unsupported type %s, not (a slice of slices) or (a slice of string) or (a function and error)", typs[0])
}

func (g *gen) typeStrings(typs []types.Type) []string {
	ss := make([]string, len(typs))
	for i := range typs {
		ss[i] = g.TypeString(typs[i])
	}
	return ss
}

func wrap(s string) string {
	if strings.Contains(s, ",") {
		return "(" + s + ")"
	}
	return s
}

func vars(prefix string, num int) []string {
	ss := make([]string, num)
	for i := range ss {
		ss[i] = prefix + strconv.Itoa(i)
	}
	return ss
}

func zip(ss, rr []string) []string {
	qq := make([]string, len(ss))
	for i := range ss {
		qq[i] = ss[i] + " " + rr[i]
	}
	return qq
}

func (g *gen) genError(typs []types.Type) error {
	p := g.printer
	g.Generating(typs...)
	name := g.GetFuncName(typs...)
	as, bs, cs, err := g.errorType(name, typs)
	if err != nil {
		return err
	}
	ats, bts, cts := g.typeStrings(as), g.typeStrings(bs), g.typeStrings(cs)
	bterrs := append(append([]string{}, bts...), "error")
	cterrs := append(append([]string{}, cts...), "error")
	a, b, c := strings.Join(ats, ", "), strings.Join(bterrs, ", "), strings.Join(cterrs, ", ")
	p.P("")

	if len(ats) > 0 {

		p.P("func %s(f func(%s) %s, g func(%s) %s) func(%s) %s {",
			name, a, wrap(b), strings.Join(bts, ", "), wrap(c), a, wrap(c))
		p.In()

		avars := vars("a", len(ats))
		avartyps := zip(avars, ats)
		p.P("return func(%s) %s {", strings.Join(avartyps, ", "), wrap(c))
		p.In()
		bvars := vars("b", len(bts))
		bvarserr := append(append([]string{}, bvars...), "err")
		p.P("%s := f(%s)", strings.Join(bvarserr, ", "), strings.Join(avars, ", "))

		p.P("if err != nil {")
		p.In()

		zeros := make([]string, len(cs))
		for i := range cs {
			zeros[i] = derive.Zero(cs[i])
		}
		ret := append(zeros, "err")
		p.P("return %s", strings.Join(ret, ", "))

		p.Out()
		p.P("}")

		p.P("return g(%s)", strings.Join(bvars, ", "))

		p.Out()
		p.P("}")

		p.Out()
		p.P("}")

	} else {

		p.P("func %s(f func() %s, g func(%s) %s) %s {",
			name, wrap(b), strings.Join(bts, ", "), wrap(c), wrap(c))
		p.In()

		bvars := vars("b", len(bts))
		bvarserr := append(append([]string{}, bvars...), "err")
		p.P("%s := f()", strings.Join(bvarserr, ", "))

		p.P("if err != nil {")
		p.In()

		zeros := make([]string, len(cs))
		for i := range cs {
			zeros[i] = derive.Zero(cs[i])
		}
		ret := append(zeros, "err")
		p.P("return %s", strings.Join(ret, ", "))

		p.Out()
		p.P("}")

		p.P("return g(%s)", strings.Join(bvars, ", "))

		p.Out()
		p.P("}")

	}

	return nil
}
