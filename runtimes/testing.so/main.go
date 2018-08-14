package main

import (
	"fmt"
	"strconv"
	"time"

	oci "github.com/opencontainers/runtime-spec/specs-go"
)

type cmd struct {
	program   string
	arguments []string
	waitChan  chan int
}

func (command *cmd) Start() error {
	switch command.program {
	case "true":
		return nil
	case "false":
		return nil
	case "sleep":
		go func() {
			durationStr := "0"
			if len(command.arguments) > 0 {
				durationStr = command.arguments[0]
			}
			duration, _ := strconv.Atoi(durationStr)
			time.Sleep(time.Duration(duration) * time.Millisecond)
			command.waitChan <- 0
		}()
		return nil
	}
	return nil
}

func (command *cmd) Wait() error {
	switch command.program {
	case "true":
		return nil
	case "false":
		return fmt.Errorf("command `false` failed")
	case "sleep":
		<-command.waitChan
		return nil
	}
	return nil
}

// Kill is stubbed for this implementation.
func (command *cmd) Kill() error { return nil }

// Bootstrapper only looks at the args, its as simple as it gets and does
// almost nothing with the rest of the oci spec.
var Bootstrapper = func(spec oci.Spec) interface{} {
	return &cmd{
		program:   spec.Process.Args[0],
		arguments: spec.Process.Args[1:],
		waitChan:  make(chan int, 0),
	}
}
