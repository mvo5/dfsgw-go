
all: dfsgw

dfsgw: dfsgw.go
	go build dfsgw.go

clean:
	rm -rf *~ dfsgw

