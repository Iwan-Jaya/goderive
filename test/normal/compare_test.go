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

package test

import (
	"reflect"
	"testing"
)

func compare(this, that interface{}) int {
	method := reflect.ValueOf(this).MethodByName("Compare")
	res := method.Call([]reflect.Value{reflect.ValueOf(that)})
	return res[0].Interface().(int)
}

func TestCompareStructs(t *testing.T) {
	structs := []interface{}{
		&BuiltInTypes{},
		&PtrToBuiltInTypes{},
		&MapsOfSimplerBuiltInTypes{},
	}
	for _, this := range structs {
		desc := reflect.TypeOf(this).Elem().Name()
		t.Run(desc, func(t *testing.T) {
			for i := 0; i < 100; i++ {
				this = random(this)
				if want, got := 0, compare(this, this); want != got {
					t.Fatalf("want %v got %v\n this = %#v\n", want, got, this)
				}
				that := random(this)
				if reflect.DeepEqual(this, that) {
					if want, got := 0, compare(this, that); want != got {
						t.Fatalf("want %v got %v\n this = %#v\n that = %#v", want, got, this, that)
					}
					if want, got := 0, compare(that, this); want != got {
						t.Fatalf("want %v got %v\n that = %#v\n this = %#v", want, got, that, this)
					}
				} else {
					c1 := compare(this, that)
					c2 := compare(that, this)
					if c1 == c2 {
						t.Fatalf("expected not equal, but got %d\n, this = %#v\n that = %#v", c1, this, that)
					}
					if c1 < 0 && c2 < 0 {
						t.Fatalf("expected not only one smaller than zero, but got %d and %d\n, this = %#v\n that = %#v", c1, c2, this, that)
					}
					if c1 > 0 && c2 > 0 {
						t.Fatalf("expected not only one bigger than zero, but got %d and %d\n, this = %#v\n that = %#v", c1, c2, this, that)
					}
				}
			}
		})
	}
}
