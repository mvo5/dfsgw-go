package main

import (
	"crypto/rand" 
	"fmt"
	"log"
	"strings"
	"html/template"
	"net/http"
	// external
	"github.com/gorilla/sessions"
	smb "github.com/mvo5/libsmbclient-go"
)

// the root smb server
const SERVER = "smb://naf1/"
const DOMAIN = "URT"

func getRandomString(length int) string {
    const alphanum = "0123456789abcdefghijklmnopqrstuvwxyz"
    var bytes = make([]byte, length)
    rand.Read(bytes)
    for i, b := range bytes {
        bytes[i] = alphanum[b % byte(len(alphanum))]
    }
    return string(bytes)
}

var session_store = sessions.NewCookieStore([]byte(getRandomString(64)))
var session_to_client_ctx = make(map[string]*smb.Client)


func handler_login(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		username := r.FormValue("username")
		password := r.FormValue("password")

		if username == "" || password == "" {
			fmt.Fprintf(w, "Need username and password\n")
			return
		}

		client := smb.New()
		fn := func(server_name, share_name string)(string, string, string) {
			return DOMAIN, username, password
		}
		client.SetAuthCallback(fn)


		dh, err := client.Opendir(SERVER)
		defer dh.Closedir()

		if err != nil {
			w.Write([]byte("Failed to login"))
			return
		}
		
		// on login, save session and client context
		session_id :=  getRandomString(64)
		// FIXME: store last access time for periodic cleanup
		session_to_client_ctx[session_id] = client

		session, _ := session_store.Get(r, "dfsgw")
		session.Values["session_id"] = session_id
		session_to_client_ctx[session_id] = client
		err = session.Save(r, w)
		if err != nil {
			fmt.Fprintf(w, "fail %s\n", err)
		}
		list_dir(w, client, dh, "/dfs")
		return
	} 
	t, _ := template.ParseFiles("login.html")
	t.Execute(w, nil)
}

func list_dir(w http.ResponseWriter, client* smb.Client, dh smb.File, parent string) {
	for {
		dirent, err := dh.Readdir()
		if err != nil {
			break
		}
		switch dirent.Type {
		case smb.SMBC_FILE_SHARE, smb.SMBC_DIR: 
			fmt.Fprintf(w, "<a href=\"%s/%s/\">%s - %s</a><p>\n", parent, dirent.Name, dirent.Name, dirent.Comment)
		case smb.SMBC_FILE: // file
			fmt.Fprintf(w, "<a href=\"%s/%s\">%s - %s</a><p>", parent, dirent.Name, dirent.Name, dirent.Comment)
		}
	}
}

func handler_logout(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "logout")
}

func handler_dfs(w http.ResponseWriter, r *http.Request) {
	filename := r.URL.Path[len("/dfs"):]
	session, _ := session_store.Get(r, "dfsgw")
	session_id, ok := session.Values["session_id"].(string)

	if !ok {
		fmt.Fprint(w, "invalid session id\n")
		return
	}

	client := session_to_client_ctx[session_id]

	// FIXME: uggggggllllllyyyy
	if strings.HasSuffix(filename, "/") {
		dh, err := client.Opendir(SERVER + filename)
		defer dh.Closedir()
		if err != nil {
			fmt.Fprintf(w, "failed to opendir %s (%s)", filename, err)
		}
		list_dir(w, client, dh, r.URL.Path)
	} else {
		f, err := client.Open(SERVER + filename, 0, 0)
		if err != nil {
			fmt.Fprintf(w, "Failed to open %s (%s)", filename, err)
		}
		defer f.Close()
		
		buf := make([]byte, 1024)
		for {
			n, err := f.Read(buf)
			if err != nil {
				fmt.Fprintf(w, "Failed to read %s (%s)", filename, err)
				break
			}
			if n == 0 {
				break
			}
			content := buf[:n]
			w.Write(content)
		}
	}
}

func main() {
	http.HandleFunc("/login", handler_login)
	http.HandleFunc("/logout", handler_logout)
	http.HandleFunc("/dfs/", handler_dfs)

	address := ":8080"
	log.Print("listen on ", address)
	http.ListenAndServe(address, nil)
}
