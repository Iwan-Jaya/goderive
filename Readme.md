# goderive

[![Build Status](https://travis-ci.org/awalterschulze/goderive.svg?branch=master)](https://travis-ci.org/awalterschulze/goderive)

`goderive` parses your go code for functions which are not implemented and then generates these functions for you by deriving their implementations from the parameter types. 

Deep Functions:

  - [Equal](http://godoc.org/github.com/awalterschulze/goderive/plugin/equal) `deriveEqual(T, T) bool`
  - [Compare](http://godoc.org/github.com/awalterschulze/goderive/plugin/compare) `deriveCompare(T, T) int`
  - [CopyTo](http://godoc.org/github.com/awalterschulze/goderive/plugin/copyto) `deriveCopyTo(src *T, dst *T)`

Tool Functions:

  - [Keys](http://godoc.org/github.com/awalterschulze/goderive/plugin/keys) `deriveKeys(map[K]V) []K`
  - [Sort](http://godoc.org/github.com/awalterschulze/goderive/plugin/sort) `deriveSort([]T) []T`
  - [Set](http://godoc.org/github.com/awalterschulze/goderive/plugin/set) `deriveSet([]T) map[T]struct{}`
  - [Min](http://godoc.org/github.com/awalterschulze/goderive/plugin/min) `deriveMin(list []T, default T) (min T)`
  - [Max](http://godoc.org/github.com/awalterschulze/goderive/plugin/max) `deriveMax(list []T, default T) (max T)`

Functional Functions:

  - [Fmap](http://godoc.org/github.com/awalterschulze/goderive/plugin/fmap) `deriveFmap(f(A) B, []A) []B`
  - [Join](http://godoc.org/github.com/awalterschulze/goderive/plugin/join) `deriveJoin([][]T) []T`

When goderive walks over your code it is looking for a function that:
  - was not implemented (or was previously derived) and
  - has a prefix predefined prefix.

Functions which have been previously derived will be regenerated to keep them up to date with the latest modifications to your types.  This keeps these functions, which are truly mundane to write, maintainable.  

For example when someone in your team adds a new field to a struct and forgets to update the CopyTo method.  This problem is solved by goderive, by generating generated functions given the new types.

Function prefixes are by default `deriveCamelCaseFunctionName`, for example `deriveEqual`.
These are customizable using command line flags.

Let `goderive` edit your function names in your source code, by enabling `autoname` and `dedup` using the command line flags.
These flags respectively makes sure than your functions have unique names and that you don't generate multiple functions that do the same thing.

## Example

In the following code the `deriveEqual` function will be spotted as a function that was not implemented (or was previously derived) and has a prefix `deriveEqual`.

```go
package main

type MyStruct struct {
	Int64     int64
	StringPtr *string
}

func (this *MyStruct) Equal(that *MyStruct) bool {
	return deriveEqual(this, that)
}
```

goderive will then generate the following code in a `derived.gen.go` file in the same package:

```go
func deriveEqual(this, that *MyStruct) bool {
	return (this == nil && that == nil) ||
		this != nil && that != nil &&
			this.Int64 == that.Int64 &&
			((this.StringPtr == nil && that.StringPtr == nil) || 
        (this.StringPtr != nil && that.StringPtr != nil && *(this.StringPtr) == *(that.StringPtr)))
}
```

## How to run

goderive can be run from the command line:

`goderive ./...`

, using the same path semantics as the go tool.

[You can also run goderive using go generate](https://github.com/awalterschulze/goderive/blob/master/example/gogenerate/example.go) 

[And you can customize function prefixes](https://github.com/awalterschulze/goderive/blob/master/example/customprefix/Makefile)

If you want you can let goderive rename your functions using the `-autoname` and `-dedup` flags.  If these flags are not used goderive will not touch your code and rather return an error.



