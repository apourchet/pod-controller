default: bins

bins: bins/controller bins/testing.so bins/shellout.so

bins/controller: $(wildcard *.go) $(wildcard ./cmd/*.go)
	go build -o bins/controller cmd/*.go

bins/%: $(wildcard runtimes/**/*.go)
	go build -buildmode=plugin -o $@ runtimes/$(notdir $@)/*.go

.PHONY: test coverage-browse clean
test: bins/testing.so bins/shellout.so
	go test -v --coverprofile=cover.out ./...

coverage-browse: test
	go tool cover --html=cover.out

clean:
	 rm bins/*
