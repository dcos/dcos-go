SUBDIRS = $(shell go list ./... | grep -v /vendor/)

test: $(SUBDIRS)

install-deps:
	go get -u github.com/golang/lint/golint

$(SUBDIRS):
	golint -set_exit_status $@
	go test -race -cover -test.v $@
