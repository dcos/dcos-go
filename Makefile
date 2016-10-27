SUBDIRS = $(shell go list ./... | grep -v /vendor/)
all: install-deps $(SUBDIRS)

test: $(SUBDIRS)

install-deps:
	go get -u github.com/golang/lint/golint
	go get -u github.com/kardianos/govendor
	govendor init
	govendor fetch github.com/dgrijalva/jwt-go@v2.6.0

$(SUBDIRS):
	golint -set_exit_status $@
	go test -race -cover -test.v $@
