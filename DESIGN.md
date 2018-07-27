// ------------------------------- Checks
// A check is a simple interface that will be easily mocked for testing purposes. It mirrors almost
// exactly what Kubernetes calls an "Action" in its API, and only has a "run" function associated
// with it.
type Check interface {
    Run() (success bool, err error)
}

type ShellCheck struct {
    exec.Command
}

type HTTPCheck struct {
    Host string
    Scheme string
    Path string
    Headers []HTTPHeader
}

type HTTPHeader struct {
    Name string
    Value string
}

// TODO: Add TCPSocket (https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.11/#probe-v1-core)

// ------------------------------- Probes
// Probes will run a Check every so often depending on what type of probe
// we are dealing with. Probes will also have the retry logic and the timeout
// logic in them so that the Check interface remains stupid simple.
type Probe interface {
    Start()
    Healthy() (healthy bool, err error)
    Running() bool
    Stop()
}

type BaseProbe struct {
    InitialDelay int
    Period int
    Timeout int
    SuccessThreshold int
    FailureThreshold int
}

// A liveness probe will continue performing the same operation at an
// interval, and will only stop if FailureThreshold is reached.
type LivenessProbe struct {
    BaseProbe
    // TODO: Other fields
}

// A readiness probe performs a check until its its SuccessThreshold
// or FailureThreshold have been met.
// The readiness probe stops running after it succeeds or fails.
type ReadinessProbe struct {
    BaseProbe
    // TODO: Other fields
}

// ExitProbe will essentially return HEALTHY while it's running,
// and HEALTHY if the exit code of the process was 0. If the exit
// code was not 0 it will return UNHEALTHY forever.
// The exitprobe will never go back to a healthy state after
// reaching an unhealthy state and stopping.
type ExitProbe struct {
    BaseProbe
    // TODO: Other fields
}

// ------------------------------- Runtime plugin
// The pod controller will load in a `process runtime` through the golang plugins
// medium with a .so file somewhere on disk. That plugin will be responsible for providing
// a few symbols, namely:
// - var Bootstrapper ctrl.ProcessBootstrapper
// - var ShellProxy ctrl.ShellProxy (optional)
// - var HTTPProxy ctrl.HTTPProxy (optional)

// In the controller package (aliased above as ctrl) we will have the following type
// declarations:
// https://github.com/opencontainers/runtime-spec/blob/master/specs-go/config.go
type ProcessBootstrapper func(args oci.Spec) (pname string, cmd Command, err error)

type Command interface {
    Start() error
    Wait() error
    Kill() error
}

type ShellProxy func(pname string, check ShellCheck) Check

type HTTPProxy func(pname string, check HTTPCheck) Check

// ------------------------------- PodController and process states
// The pod controller will need to keep track of the states of the processes it manages. It will
// mostly only really need the set of probes that match each process and will simply aggregate
// the information from the probes into a singular state per process.
// A second pass will take the states of each of the processes and aggregate those into a single
// HEALTHY bit of information.
type ProcessState int

const (
    Known ProcessState = iota
    Started // When the process was started but the readiness probe has not been successful yet
    Unready // When the readiness probe has given up
    Healthy // When the readiness and liveness probe are healthy
    Unhealthy // When the liveness probe last failed after a healthy state
    Terminal // When the liveness probe has given up
    Finished // When the process exited with a 0 status code
    Failed // When the process exited with non-0 status code
)


type ProcessStatus struct {
    State ProcessState
    Restarts int
    LatestErrors []error
}

type PodController interface {
    Start() error
    Status() []ProcessStatus
    Healthy() bool // False should cause the pod to be restarted
}
