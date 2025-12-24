package run

import (
	"github.com/spf13/afero"
)

type Controller struct {
	fs    afero.Fs
	param *ParamRun
}

func New(fs afero.Fs, param *ParamRun) *Controller {
	return &Controller{
		param: param,
		fs:    fs,
	}
}
