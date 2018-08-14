package main

import (
	"fmt"
	"os/exec"
	"strconv"
	"syscall"
	"time"

	oci "github.com/opencontainers/runtime-spec/specs-go"
)

type container struct {
	program   string
	arguments []string
	waitChan  chan int
}

func (ctn *container) Start() error {
	switch ctn.program {
	case "true":
		return nil
	case "false":
		return nil
	case "sleep":
		go func() {
			durationStr := "0"
			if len(ctn.arguments) > 0 {
				durationStr = ctn.arguments[0]
			}
			duration, _ := strconv.Atoi(durationStr)
			time.Sleep(time.Duration(duration) * time.Millisecond)
			ctn.waitChan <- 0
		}()
		return nil
	}
	return nil
}

func (ctn *container) Wait() error {
	switch ctn.program {
	case "true":
		return nil
	case "false":
		return fmt.Errorf("command `false` failed")
	case "sleep":
		<-ctn.waitChan
		return nil
	}
	return nil
}

// Kill is stubbed for this implementation.
func (ctn *container) Kill() error { return nil }

// Exec just executes the command on the host.
func (ctn *container) Exec(program string, arguments ...string) (code int, err error) {
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
	return &container{
		program:   spec.Process.Args[0],
		arguments: spec.Process.Args[1:],
		waitChan:  make(chan int, 0),
	}
}
