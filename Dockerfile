FROM golang:1.10 as builder
ENV GOBIN /go/bin
RUN go get -u github.com/golang/dep/...
ADD . /go/src/github.com/apourchet/pod-controller
WORKDIR /go/src/github.com/apourchet/pod-controller
RUN make clean bins

FROM debian:8
COPY --from=builder /go/src/github.com/apourchet/pod-controller/bins /bins
ENTRYPOINT ["/bins/controller"]
