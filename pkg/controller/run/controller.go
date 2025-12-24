package run

import (
	"github.com/spf13/afero"
	"github.com/suzuki-shunsuke/rgo/pkg/cmdexec"
)

type Controller struct {
	fs    afero.Fs
	param *ParamRun
	exec  *cmdexec.Executor
}

func New(fs afero.Fs, param *ParamRun, exec *cmdexec.Executor) *Controller {
	return &Controller{
		param: param,
		fs:    fs,
		exec:  exec,
	}
}
