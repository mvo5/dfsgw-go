Generic smb to http gateway
===========================

This is a generic smb to http gateway (readonly for now).  You need to
login via your ActiveDirectory name and it will list the content of
the selected windows share. 

To test against the real Uni Trier server run inside the Uni network
(or via VPN):
```
$ ./dfsgw -server smb://naf1 -domain URT
2013/11/09 22:12:19 Using server smb://naf1 (domain URT)
2013/11/09 22:12:19 listen on :8080
$ firefox http://localhost:8080/login
```


Build:
```
$ sudo apt-get install golang libsmbclient-dev
$ . env.sh
$ go get github.com/mvo5/libsmbclient-go
$ go get github.com/gorilla/sessions 
```

To install the css/js:
```
$ npm install bower
$ ./node_modules/.bin/bower install
```

