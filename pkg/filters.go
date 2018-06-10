package pkg

import (
	"github.com/flosch/pongo2"
	. "github.com/flosch/pongo2"
	"path"
)

func init() {
	pongo2.RegisterFilter("basename", basename)
	pongo2.RegisterFilter("dirname", dirname)

}

func basename(in *Value, param *Value) (*Value, *Error) {
	if in.IsNil() {
		return param, nil
	}

	return AsValue(path.Base(in.String())), nil
}

func dirname(in *Value, param *Value) (*Value, *Error) {
	if in.IsNil() {
		return param, nil
	}
	return AsValue(path.Dir(in.String())), nil
}
