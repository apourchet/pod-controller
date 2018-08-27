default: bins

RUNTIMES := $(shell bash -c 'ls runtimes | while read line; do echo bins/$$line; done')

bins: bins/controller $(RUNTIMES)

bins/controller: vendor $(wildcard *.go) $(wildcard ./cmd/*.go)
	mkdir -p bins
	CGO_ENABLED=1 go build -gcflags '-N -l' -o bins/controller cmd/*.go

bins/%: $(wildcard runtimes/**/*.go)
	mkdir -p bins
	CGO_ENABLED=1 go build -gcflags '-N -l' -buildmode=plugin -o $@ runtimes/$(notdir $@)/*.go

vendor: Gopkg.toml
	dep ensure


.PHONY: ctest test unit-test integration coverage-browse clean

test:
	docker build -f dockerfiles/unit-test.df -t pc-unit .
	$(MAKE) integration

unit-test: vendor $(RUNTIMES)
	go test -v --coverprofile=cover.out ./...

coverage-browse: unit-test
	go tool cover --html=cover.out

integration: bins
	pytest ./tests/python/

clean:
	 rm -rf bins/*

# TODO: remove temp targets
.PHONY: demo demo-watch demo-kill
demo:
	docker build -t pod-controller .
	docker run -it --rm \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v $(shell pwd)/tests/specs/healthy_double_forever.json:/spec.json \
		-p 8888:8888 \
		pod-controller --runtime /bins/docker-simple.so

demo-watch:
	watch 'docker ps && echo "" && curl -s localhost:8888/healthy && echo "" && curl -s localhost:8888/status | jq .'

demo-kill:
	curl localhost:8888/kill
