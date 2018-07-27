package controller

import (
	"plugin"

	oci "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/pkg/errors"
)

// The pod controller will load in a `process runtime` through the golang plugins
// medium with a .so file somewhere on disk. That plugin will be responsible for providing
// a few symbols, namely:
// - var Bootstrapper ctrl.ContainerBootstrapper
// - var ShellProxy ctrl.ShellProxy (optional)
// - var HTTPProxy ctrl.HTTPProxy (optional)

// In the controller package (aliased above as ctrl) we will have the following type
// declarations:
// https://github.com/opencontainers/runtime-spec/blob/master/specs-go/config.go
type ContainerBootstrapper func(args oci.Spec) (pname string, cmd Command, err error)

type Command interface {
	Start() error
	Wait() error
	Kill() error
}

type ShellProxy func(pname string, check ShellCheck) Check

type HTTPProxy func(pname string, check HTTPCheck) Check

type RuntimeStrategy struct {
	Bootstrapper ContainerBootstrapper
	ShellProxy   ShellProxy
	HTTPProxy    HTTPProxy
}

// LoadPlugin takes in a path to a .so file and returns a RuntimeStrategy that can be
// used to launch and watch containers.
func LoadPlugin(path string) (RuntimeStrategy, error) {
	strategy := RuntimeStrategy{}
	p, err := plugin.Open(path)
	if err != nil {
		return strategy, errors.WithStack(err)
	}

	bts, err := p.Lookup("Bootstrapper")
	if err != nil {
		return strategy, errors.New("Bootstrapper is a mandatory part of the runtime")
	}
	strategy.Bootstrapper = bts.(ContainerBootstrapper)

	shellProxy, err := p.Lookup("ShellProxy")
	if err != nil {
		shellProxy = func(_ string, check ShellCheck) Check { return check }
	}
	strategy.ShellProxy = shellProxy.(ShellProxy)

	httpProxy, err := p.Lookup("HTTPProxy")
	if err != nil {
		httpProxy = func(_ string, check HTTPCheck) Check { return check }
	}
	strategy.HTTPProxy = httpProxy.(HTTPProxy)

	return strategy, nil
}

// CheckFromCommand takes a command returned by the ContainerBootstrapper and returns a Check
// that syncronously Starts and Waits.
func CheckFromCommand(command Command) Check {
	return ShellCheck{
		Runner: func() error {
			if err := command.Start(); err != nil {
				return err
			}
			return command.Wait()
		},
	}
}
