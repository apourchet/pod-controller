package main

import (
	"local/controller"
	"os/exec"

	oci "github.com/opencontainers/runtime-spec/specs-go"
)

type cmd struct {
	*exec.Cmd
}

func (command cmd) Kill() error { return command.Process.Kill() }

// This bootstrapper only looks at the args, its as simple as it gets and does
// almost nothing with the rest of the oci spec.
var Bootstrapper = func(spec oci.Spec) (string, controller.Command, error) {
	command := exec.Command(spec.Process.Args[0], spec.Process.Args[1:]...)
	return "", &cmd{command}, nil
}
