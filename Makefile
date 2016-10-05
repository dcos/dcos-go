SUBDIRS = $(shell go list ./... | grep -v /vendor/)

all: $(SUBDIRS)

$(SUBDIRS):
	go get -u github.com/golang/lint/golint
	golint -set_exit_status $@
	go test -race -cover -test.v $@

.PHONY: all $(SUBDIRS)

