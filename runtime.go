package controller

import (
	"fmt"
	"plugin"

	oci "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/pkg/errors"
)

// The pod controller will load in a `process runtime` through the golang plugins
// medium with a .so file somewhere on disk. That plugin will be responsible for providing
// the following symbols:
// - var Bootstrapper func(spec oci.Spec) interface{}

// In the controller package (aliased above as ctrl) we will have the following type
// declarations:
// https://github.com/opencontainers/runtime-spec/blob/master/specs-go/config.go
type ContainerBootstrapper func(args oci.Spec) (Container, error)

type Container interface {
	Start() error
	Wait() error
	Kill() error
}

type RuntimeStrategy struct {
	Bootstrapper ContainerBootstrapper
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

	// Ensure that the bootstrapper function has the right type.
	if _, ok := bts.(*func(spec oci.Spec) interface{}); !ok {
		return strategy, fmt.Errorf("Type check failed for runtime bootstrapper in plugin %s", path)
	}

	strategy.Bootstrapper = func(spec oci.Spec) (Container, error) {
		fn := bts.(*func(spec oci.Spec) interface{})
		val := (*fn)(spec)
		if err, ok := val.(error); ok {
			return nil, err
		} else if ctn, ok := val.(Container); ok {
			return ctn, nil
		}
		return nil, fmt.Errorf("Failed to cast return value of bootstrapper to Container")
	}

	return strategy, nil
}

// CheckFromContainer takes a command returned by the ContainerBootstrapper and returns a Check
// that syncronously Starts and Waits.
func CheckFromContainer(ctn Container) Check {
	return RunnerCheck{
		Runner: func() error {
			if err := ctn.Start(); err != nil {
				return err
			}
			return ctn.Wait()
		},
	}
}
