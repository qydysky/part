package testdata

import (
	eg "github.com/qydysky/part/component/testdata/eg"

	comp "github.com/qydysky/part/component"
)

func init() {
	var linkMap = map[string][]string{
		`test`: {
			comp.Sign[eg.Sign](),
		},
	}
	comp.Link(linkMap)
}
