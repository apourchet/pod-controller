package main

import (
	"os/exec"
	"syscall"

	oci "github.com/opencontainers/runtime-spec/specs-go"
)

type container struct {
	cmd *exec.Cmd
}

func (ctn *container) Start() error { return ctn.cmd.Start() }

func (ctn *container) Wait() error { return ctn.cmd.Wait() }

func (ctn *container) Kill(signal int) error { return ctn.cmd.Process.Kill() }

func (ctn *container) Exec(program string, arguments ...string) (code int, err error) {
	command := exec.Command(program, arguments...)
	err = command.Run()
	if err == nil {
		return 0, nil
	} else if exiterr, ok := err.(*exec.ExitError); ok {
		if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
			return status.ExitStatus(), err
		}
	}
	return 1, err
}

// Bootstrapper only looks at the args, its as simple as it gets and does
// almost nothing with the rest of the oci spec.
var Bootstrapper = func(spec oci.Spec, meta map[string]interface{}) (interface{}, error) {
	command := exec.Command(spec.Process.Args[0], spec.Process.Args[1:]...)
	return &container{cmd: command}, nil
}
