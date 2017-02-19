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
	"go/parser"
	"go/types"
	"strconv"

	"golang.org/x/tools/go/loader"
)

func load(paths ...string) (*loader.Program, error) {
	conf := loader.Config{
		ParserMode:  parser.ParseComments,
		AllowErrors: true,
	}
	rest, err := conf.FromArgs(paths, true)
	if err != nil {
		return nil, fmt.Errorf("could not parse arguments: %s", err)
	}
	if len(rest) > 0 {
		return nil, fmt.Errorf("unhandled extra arguments: %v", rest)
	}
	p, err := conf.Load()
	if err != nil {
		return nil, err
	}
	if p.Fset == nil {
		return nil, fmt.Errorf("program == nil")
	}
	return p, nil
}

func typeName(typ types.Type, qual types.Qualifier) string {
	switch t := typ.(type) {
	case *types.Pointer:
		return "PtrTo" + typeName(t.Elem(), qual)
	case *types.Array:
		sizeStr := strconv.Itoa(int(t.Len()))
		return "Array" + sizeStr + "Of" + typeName(t.Elem(), qual)
	case *types.Slice:
		return "SliceOf" + typeName(t.Elem(), qual)
	case *types.Map:
		return "MapOf" + typeName(t.Key(), qual) + "To" + typeName(t.Elem(), qual)
	}
	return types.TypeString(typ, qual)
}
