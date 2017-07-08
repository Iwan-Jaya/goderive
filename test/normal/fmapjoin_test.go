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
	"strings"
	"testing"
)

func TestFmap(t *testing.T) {
	got := deriveFmap(func(i int) int { return i + 1 }, []int{1, 2})
	want := []int{2, 3}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestFmapString(t *testing.T) {
	got := deriveFmapString(func(r rune) bool { return r == 'a' }, "abc")
	want := []bool{true, false, false}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestJoin(t *testing.T) {
	got := deriveJoin([][]int{{1, 2}, {3, 4}})
	want := []int{1, 2, 3, 4}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestFmapJoin(t *testing.T) {
	ss := []string{"a,b", "c,d"}
	split := func(s string) []string {
		return strings.Split(s, ",")
	}
	got := deriveJoinSS(deriveFmapSS(split, ss))
	want := []string{"a", "b", "c", "d"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}
