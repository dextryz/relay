.PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

# ------------------------------------------------------------------------------------ #
# QUALITY CONTROL
# ------------------------------------------------------------------------------------ #

## tidy: format code and tidy modfile
.PHONY: tidy
tidy:
	go fmt ./...
	go mod tidy -v

## audit: run quality control checks
.PHONY: audit
audit:
	go vet ./...

# ------------------------------------------------------------------------------------ #
# DEVELOPMENT
# ------------------------------------------------------------------------------------ #

## test: run all tests
.PHONY: test
test:
	go test -v -race -buildvcs ./...# test: run all tests

## build: build the application
build:
	go build
