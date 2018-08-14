package main

import (
	"os/exec"
	"syscall"

	oci "github.com/opencontainers/runtime-spec/specs-go"
)

type cmd struct {
	*exec.Cmd
}

// Kill is stubbed for this implementation.
func (command *cmd) Kill() error { return command.Process.Kill() }

func (command *cmd) Exec(program string, arguments ...string) (code int, err error) {
	cmd := exec.Command(program, arguments...)
	err = cmd.Run()
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
var Bootstrapper = func(spec oci.Spec) interface{} {
	command := exec.Command(spec.Process.Args[0], spec.Process.Args[1:]...)
	return &cmd{command}
}
