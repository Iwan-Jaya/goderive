// Example: customprefix shows how to defined a derived function that does not
// have to start with default "deriveEqual" prefix.
// in the Makefile we can see the goderive command being called:
//
//   goderive --equal.prefix="eq" ./...
//
// This sets the new prefix to "eq".
package customprefix

type MyStruct struct {
	Int64     int64
	StringPtr *string
}

func (this *MyStruct) Equal(that *MyStruct) bool {
	return eq(this, that)
}
