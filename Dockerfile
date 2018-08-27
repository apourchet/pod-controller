FROM golang:1.10 as builder
ENV GOBIN /go/bin
RUN go get -u github.com/golang/dep/...
ADD . /go/src/code.uber.internal/personal/pourchet/pod-controller
WORKDIR /go/src/code.uber.internal/personal/pourchet/pod-controller
RUN make clean bins

FROM debian:8
COPY --from=builder /go/src/code.uber.internal/personal/pourchet/pod-controller/bins /bins
ENTRYPOINT ["/bins/controller"]
