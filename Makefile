
all: dfsgw

dfsgw: dfsgw.go packages
	export GOPATH=`pwd`; go build dfsgw.go

packages:
	export GOPATH=`pwd`; go get github.com/mvo5/libsmbclient-go
	export GOPATH=`pwd`; go get github.com/gorilla/sessions

clean:
	rm -rf *~ dfsgw pkg/ src/

