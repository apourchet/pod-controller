package main

import (
	"os/exec"

	oci "github.com/opencontainers/runtime-spec/specs-go"
)

type cmd struct {
	*exec.Cmd
}

// Kill is stubbed for this implementation.
func (command *cmd) Kill() error { return command.Process.Kill() }

// Bootstrapper only looks at the args, its as simple as it gets and does
// almost nothing with the rest of the oci spec.
var Bootstrapper = func(spec oci.Spec) interface{} {
	command := exec.Command(spec.Process.Args[0], spec.Process.Args[1:]...)
	return &cmd{command}
}
