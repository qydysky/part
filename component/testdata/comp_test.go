package testdata

import (
	"context"
	"testing"

	comp "github.com/qydysky/part/component"
)

func TestMain(t *testing.T) {
	var s = "s"
	if e := comp.Run[string](`test`, context.Background(), &s); e != nil {
		t.Fatal(e)
	}
}
