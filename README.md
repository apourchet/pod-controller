 # Pod Controller
 ---------------
 
 ## Purpose
 The pod controller is meant to be used as a library to launch and monitor containers. The controller relies on a runtime plugin to define the containerization strategy, and a pod spec that will be materialized into its list of containers when the controller is started. It takes aggregates the states of the containers in the pod into a single `healthy` bit that can then be thought of as the health of the pod as a whole.
 
 ## Pod Spec Example
 The following pod spec has a single init container that will run before the pod starts. Each of the containers in the `initContainers` field will run sequentially and in that order, until they terminate. In this example the init container just sleeps for 10 seconds. The pod then contains 2 containers that sleep for 1000 and 1001 seconds respectively after touching a file `/tmp/health`. Their health checks (liveness) will simply try to cat that file every 2 seconds with a timeout of 1 second. The schema for the liveness probe is the same as in Kubernetes and aims to have the exact same behavior. These two containers will be run simultaneously and be health checked in the background. 
 ```json
{
    "initContainers": [
        {
            "process": {
                "args": ["sleep", "10"],
            }
        }
    ],   
    "containers": [
        {
            "name": "main",
            "process": {
                "args": ["/bin/sh", "-c", "touch /tmp/health && sleep 1000"]
            },
            "livenessProbe": {
                "exec": ["/bin/sh", "-c", "sleep 1 && cat /tmp/health"],
                "initialDelaySeconds": 1,
                "periodSeconds": 2,
                "timeoutSeconds": 2,
                "successThreshold": 1,
                "failureThreshold": 30
            }
        },
        {
            "name": "sidecar",
            "process": {
                "args": ["/bin/sh", "-c", "touch /tmp/health && sleep 1001"]
            },
            "livenessProbe": {
                "exec": ["/bin/sh", "-c", "sleep 1 && cat /tmp/health"],
                "initialDelaySeconds": 1,
                "periodSeconds": 2,
                "timeoutSeconds": 2,
                "successThreshold": 1,
                "failureThreshold": 1
            }
        }
    ]
}
```

## Runtime Plugin Example
The pod controller does not come with any production-ready containerization strategies, instead requiring a `.so` plugin to be wired in. The following is a dummy plugin to show what functions should be provided. 
```go
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
var Bootstrapper = func(spec oci.Spec) interface{} {
	command := exec.Command(spec.Process.Args[0], spec.Process.Args[1:]...)
	return &container{cmd: command}
}
```
The plugin only needs to define the `Bootstrapper` function that takes in an oci.Spec and returns either either an `error` or a `controller.Container` interface:
```go
type Container interface {
	Start() error
	Wait() error
	Kill(signal int) error
	Exec(program string, arguments ...string) (int, error)
}
```

## Demonstration
As a demonstration we wrote a simple http server that will output as JSON the outputs of `Healthy()` and `Status()` of the controller. To run the demo you need to have docker installed and the socket to the daemon should be located at `/var/run/docker.sock`. 

To start the demo, run in a session: `make demo`. Then in another session run `make demo-watch`, you should notice that two new containers got started by the pod controller through a simple docker runtime plugin (located at `runtimes/docker-simple.so/main.go`). The pod controller is actively health checking those two containers and the output of `watch` contains the JSONified values of `Healthy()` and `Status()`. If you exec into one of the two debian containers and remove `/tmp/health` you will see that the container will start failing (after the failure threshold has been reached) and the health bit of the pod will flip to false.
![demo](https://user-images.githubusercontent.com/2396687/44236871-56821500-a163-11e8-9324-b8600d6e41b6.gif)
