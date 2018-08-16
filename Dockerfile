FROM golang:1.10 as builder
ADD . /go/src/local/controller
WORKDIR /go/src/local/controller
RUN make clean bins

FROM debian:8
COPY --from=builder /go/src/local/controller/bins /bins
ENTRYPOINT ["/bins/controller"]
