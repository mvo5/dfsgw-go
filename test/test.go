package main

import (
	"net/http"
	"net/url"
	"fmt"
	"log"
	"io/ioutil"
	"sync"
)

// from http://stackoverflow.com/questions/12756782/go-http-post-and-use-cookies
type Jar struct {
    lk      sync.Mutex
    cookies map[string][]*http.Cookie
}
func NewJar() *Jar {
    jar := new(Jar)
    jar.cookies = make(map[string][]*http.Cookie)
    return jar
}
func (jar *Jar) SetCookies(u *url.URL, cookies []*http.Cookie) {
    jar.lk.Lock()
    jar.cookies[u.Host] = cookies
    jar.lk.Unlock()
}
func (jar *Jar) Cookies(u *url.URL) []*http.Cookie {
    return jar.cookies[u.Host]
}


var baseuri = "http://localhost:8080/"

func login(client *http.Client) {
	resp, err := client.PostForm(baseuri + "login",
		url.Values{"username": {"x"}, "password": {"x"}})
	if err != nil {
		log.Fatal(err)
	}
	showBody(resp)
}

func showBody(resp *http.Response) {
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(resp.Request.URL)
	fmt.Println(string(body))
	fmt.Println("--------------------------")
}

func readPage(client *http.Client, url string) {
	resp, err := client.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	showBody(resp)
}


func main() {
	ch := make(chan int)
	count := 0
	THREADS := 10

	jar := NewJar()
	client := http.Client{nil, nil, jar}
	login(&client)

	for i := 0; i < THREADS; i++ {
		go func() {
			readPage(&client, baseuri + "dfs/test/")
			ch <- 1
		}()
		go func() {
			readPage(&client, baseuri + "dfs/test/archive.ubuntu.com_ubuntu_dists_saucy_universe_binary-amd64_Packages")
			ch <- 1
		}()
	}
	
	for count < 2*THREADS {
		count += <- ch
	}
}
