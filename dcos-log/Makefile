all: lint test build

gofmt:
	@echo "+$@"
	go fmt ./...
lint:
	@echo "+$@"
	go get -u github.com/golang/lint/golint
	golint -set_exit_status ./...

test:
	@echo "+$@"

build:
	@echo "+$@"
	go build

clean:
	@echo "+$@"
	rm -f dcos-log
