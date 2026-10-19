package dotenv

import (
	"reflect"
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	in := `
# comment
PLAIN=value
export EXPORTED=yes
SPACED = padded
INLINE=abc # trailing comment
SINGLE='literal $HOME # kept'
DOUBLE="line1\nline2"
EMPTY=
URL=postgres://u:p@host/db?x=1
not a pair
`
	got, err := Parse(strings.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"PLAIN":    "value",
		"EXPORTED": "yes",
		"SPACED":   "padded",
		"INLINE":   "abc",
		"SINGLE":   "literal $HOME # kept",
		"DOUBLE":   "line1\nline2",
		"EMPTY":    "",
		"URL":      "postgres://u:p@host/db?x=1",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v\nwant %#v", got, want)
	}
}
