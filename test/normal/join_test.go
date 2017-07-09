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
	"errors"
	"reflect"
	"testing"
)

func TestJoin(t *testing.T) {
	got := deriveJoin([][]int{{1, 2}, {3, 4}})
	want := []int{1, 2, 3, 4}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestJoinString(t *testing.T) {
	got := deriveJoinString([]string{"abc", "cde"})
	want := "abccde"
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestJoinJustErrors(t *testing.T) {
	t.Run("no error", func(t *testing.T) {
		f := func() error {
			return nil
		}
		var myerr error
		err := deriveJoinJustError(f, myerr)
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("first error", func(t *testing.T) {
		f := func() error {
			return nil
		}
		var myerr error = errors.New("a")
		err := deriveJoinJustError(f, myerr)
		if err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("second error", func(t *testing.T) {
		f := func() error {
			return errors.New("a")
		}
		var myerr error
		err := deriveJoinJustError(f, myerr)
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestJoinErrorAndValue(t *testing.T) {
	t.Run("no error", func(t *testing.T) {
		want := "a"
		f := func() (string, error) {
			return want, nil
		}
		var myerr error
		got, err := deriveJoinErrorAndString(f, myerr)
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Fatalf("got %s != want %s", got, want)
		}
	})
	t.Run("first error", func(t *testing.T) {
		f := func() (string, error) {
			return "a", nil
		}
		var myerr error = errors.New("a")
		_, err := deriveJoinErrorAndString(f, myerr)
		if err == nil {
			t.Fatal("expected error")
		}
	})
	t.Run("second error", func(t *testing.T) {
		f := func() (string, error) {
			return "a", errors.New("a")
		}
		var myerr error
		_, err := deriveJoinErrorAndString(f, myerr)
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestJoinErrorAndValues(t *testing.T) {
	want := "a"
	wanti := 1
	f := func() (string, int, error) {
		return want, wanti, nil
	}
	var myerr error
	got, goti, err := deriveJoinErrorAndValues(f, myerr)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("got %s != want %s", got, want)
	}
	if goti != wanti {
		t.Fatalf("got %d != want %d", got, want)
	}
}
