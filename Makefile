default: bins

RUNTIMES := $(shell bash -c 'ls runtimes | while read line; do echo bins/$$line; done')

bins: bins/controller $(RUNTIMES)

bins/controller: vendor $(wildcard *.go) $(wildcard ./cmd/*.go)
	CGO_ENABLED=1 go build -gcflags '-N -l' -o bins/controller cmd/*.go

bins/%: $(wildcard runtimes/**/*.go)
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
	 rm bins/*
