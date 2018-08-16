package main

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/api/types"
	docker_container "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"

	oci "github.com/opencontainers/runtime-spec/specs-go"
)

type container struct {
	ID string

	image     string
	program   string
	arguments []string
	client    *client.Client
}

func (ctn *container) Start() error {
	config := &docker_container.Config{
		Image: ctn.image,
		Cmd:   append([]string{ctn.program}, ctn.arguments...),
	}
	resp, err := ctn.client.ContainerCreate(context.Background(), config, nil, nil, "")
	if err != nil {
		fmt.Println("error creating", err)
		return err
	}
	ctn.ID = resp.ID

	if err := ctn.client.ContainerStart(context.Background(), ctn.ID, types.ContainerStartOptions{}); err != nil {
		fmt.Println("error starting", err)
		return err
	}
	return nil
}

func (ctn *container) Wait() error {
	code, err := ctn.client.ContainerWait(context.Background(), ctn.ID)
	if err != nil {
		return err
	} else if code != 0 {
		return fmt.Errorf("non-zero exit code from wait: %d", code)
	}
	return nil
}

func (ctn *container) Kill(signal int) error {
	return ctn.client.ContainerKill(context.Background(), ctn.ID, "9")
}

func (ctn *container) Exec(program string, arguments ...string) (code int, err error) {
	config := types.ExecConfig{
		Cmd: append([]string{program}, arguments...),
	}
	resp, err := ctn.client.ContainerExecCreate(context.Background(), ctn.ID, config)
	if err != nil {
		fmt.Println("ExecCreate error", err)
		return 1, err
	}
	err = ctn.client.ContainerExecStart(context.Background(), resp.ID, types.ExecStartCheck{})
	if err != nil {
		fmt.Println("ExecStart error", err)
		return 1, err
	}

	for {
		inspect, err := ctn.client.ContainerExecInspect(context.Background(), resp.ID)
		if err != nil {
			fmt.Println("ExecInspect error", err)
			return 1, err
		} else if !inspect.Running {
			fmt.Printf("inspect: %+v\n", inspect)
			return inspect.ExitCode, nil
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// Bootstrapper only looks at the args, its as simple as it gets and does
// almost nothing with the rest of the oci spec.
var Bootstrapper = func(spec oci.Spec) interface{} {
	image := "debian:9"
	if spec.Root != nil && spec.Root.Path != "" {
		image = spec.Root.Path
	}
	cli, err := client.NewEnvClient()
	if err != nil {
		return err
	}
	return &container{
		image:     image,
		program:   spec.Process.Args[0],
		arguments: spec.Process.Args[1:],
		client:    cli,
	}
}
