export GOPATH := $(shell pwd)

all: dfsgw

dfsgw: dfsgw.go packages
	go build dfsgw.go

packages:
	go get github.com/mvo5/libsmbclient-go
	go get github.com/gorilla/sessions

clean:
	rm -rf *~ dfsgw pkg/ src/

