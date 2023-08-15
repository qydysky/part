package eg

import (
	"context"
	"errors"

	comp "github.com/qydysky/part/component"
)

type Sign struct{}

func init() {
	if e := comp.Put[string](comp.Sign[Sign](), deal); e != nil {
		panic(e)
	}
}

func deal(ctx context.Context, ptr *string) error {
	if *ptr == "s" {
		return errors.New("1")
	}
	return nil
}
