package controller

// TODO: Add TCPSocket (https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.11/#probe-v1-core)

import (
	"fmt"
	"net/http"
	"os/exec"
)

// A Check is a simple interface that will be easily mocked for testing purposes. It mirrors almost
// exactly what Kubernetes calls an "Action" in its API, and only has a "run" function associated
// with it.
type Check interface {
	Run() (success bool, err error)
}

var _ Check = HTTPCheck{}
var _ Check = ShellCheck{}
var _ Check = HealthyCheck{}

// A RunnerCheck implements the Check interface and calls the Runner function.
type RunnerCheck struct {
	Runner func() error
}

// Run implements Check.Run.
func (check RunnerCheck) Run() (bool, error) {
	if err := check.Runner(); err != nil {
		return false, err
	}
	return true, nil
}

// A ShellCheck implements the Check interface and runs a command, reporting an error if the
// exit code is not 0.
type ShellCheck struct {
	Cmd *exec.Cmd
}

// NewShellCheck returns a new ShellCheck with the provided *exec.Cmd.
func NewShellCheck(cmd *exec.Cmd) ShellCheck {
	return ShellCheck{
		Cmd: cmd,
	}
}

// Run runs the command and returns an error if the command has an error, returning
// true and no error otherwise.
func (check ShellCheck) Run() (bool, error) {
	if err := check.Cmd.Run(); err != nil {
		return false, err
	}
	return true, nil
}

// HealthyCheck always returns a healthy bit set to true and no error.
type HealthyCheck struct{}

// Run implements Check.Run.
func (HealthyCheck) Run() (bool, error) { return true, nil }

// An HTTPCheck sends a GET request to the host/path provided and checks the status code to
// determine if the check succeeded or failed.
type HTTPCheck struct {
	Scheme  string
	Host    string
	Path    string
	Headers []HTTPHeader

	Client       HTTPDoer
	SuccessCodes []int
}

type HTTPHeader struct {
	Name  string
	Value string
}

type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

var ErrBadStatusCode = fmt.Errorf("Bad Status Code")

func NewHTTPCheck(host string, path string) HTTPCheck {
	return HTTPCheck{
		Scheme:       "http",
		Host:         host,
		Path:         path,
		Client:       http.DefaultClient,
		SuccessCodes: []int{http.StatusOK},
	}
}

func (check *HTTPCheck) AddHeader(name, value string) {
	check.Headers = append(check.Headers, HTTPHeader{
		Name:  name,
		Value: value,
	})
}

func (check HTTPCheck) Run() (bool, error) {
	url := fmt.Sprintf("%s://%s%s", check.Scheme, check.Host, check.Path)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false, err
	}
	for _, header := range check.Headers {
		req.Header.Add(header.Name, header.Value)
	}

	resp, err := check.Client.Do(req)
	if err != nil {
		return false, err
	} else if !intsContain(check.SuccessCodes, resp.StatusCode) {
		return false, ErrBadStatusCode
	}
	return true, nil
}
