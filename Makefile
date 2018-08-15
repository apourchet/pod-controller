default: bins

RUNTIMES := $(shell bash -c 'ls runtimes | while read line; do echo bins/$$line; done')

bins: bins/controller $(RUNTIMES)

bins/controller: $(wildcard *.go) $(wildcard ./cmd/*.go)
	go build -o bins/controller cmd/*.go

bins/%: $(wildcard runtimes/**/*.go)
	go build -buildmode=plugin -o $@ runtimes/$(notdir $@)/*.go

.PHONY: test integration coverage-browse clean
test: $(RUNTIMES)
	go test -v --coverprofile=cover.out ./...

coverage-browse: test
	go tool cover --html=cover.out

integration: bins
	pytest ./tests/python/

clean:
	 rm bins/*
