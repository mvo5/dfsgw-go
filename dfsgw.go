package main

import (
	"crypto/rand"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"
	// external
	"github.com/gorilla/sessions"
	smb "github.com/mvo5/libsmbclient-go"
)

// the root smb server
//const SERVER = "smb://naf1/"
var SERVER = "smb://localhost/"
var DOMAIN = "URT"

func getRandomString(length int) string {
	const alphanum = "0123456789abcdefghijklmnopqrstuvwxyz"
	var bytes = make([]byte, length)
	rand.Read(bytes)
	for i, b := range bytes {
		bytes[i] = alphanum[b%byte(len(alphanum))]
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
		fn := func(server_name, share_name string) (string, string, string) {
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
		session_id := getRandomString(64)
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
	t, err := template.ParseFiles("templates/base.html",  "templates/login.html")
	if err != nil {
		log.Fatal(err)
	}
	t.ExecuteTemplate(w, "base", nil)
}

func list_dir(w http.ResponseWriter, client *smb.Client, dh smb.File, parent string) {
	type Dir struct {
		Parent   string
		Dirinfo  []smb.Dirent
		Fileinfo []smb.Dirent
	}
	d := Dir{Parent: parent}
	for {
		dirent, err := dh.Readdir()
		if err != nil {
			break
		}
		if dirent.Name == ".." || dirent.Name == "." {
			continue
		}
		switch dirent.Type {
		case smb.SMBC_FILE_SHARE, smb.SMBC_DIR:
			d.Dirinfo = append(d.Dirinfo, *dirent)
		case smb.SMBC_FILE:
			d.Fileinfo = append(d.Fileinfo, *dirent)
		}
	}
	t := template.Must(template.New("dir").ParseFiles("templates/base.html", "templates/dir.html"))
	err := t.ExecuteTemplate(w, "base", d)
	if err != nil {
		log.Fatal(err)
	}
}

func handler_logout(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("templates/base.html", "templates/logout.html")
	if err != nil {
		log.Fatal(err)
	}
	session, _ := session_store.Get(r, "dfsgw")
	session_id, ok := session.Values["session_id"].(string)
	if ok {
		client := session_to_client_ctx[session_id]
		client.Destroy()
		delete(session_to_client_ctx, session_id)

		session.Values["session_id"] = nil
		session.Save(r, w)
	}

	t.ExecuteTemplate(w, "base", nil)

}

func handler_dfs(w http.ResponseWriter, r *http.Request) {
	filename := r.URL.Path[len("/dfs"):]
	session, _ := session_store.Get(r, "dfsgw")
	session_id, ok := session.Values["session_id"].(string)

	if !ok {
		// FIXME: send proper error code
		fmt.Fprint(w, "<html>invalid session id, please <a href='/login'>login</a>\n</html>")
		return
	}

	client := session_to_client_ctx[session_id]

	// FIXME: uggggggllllllyyyy
	if strings.HasSuffix(filename, "/") {
		dh, err := client.Opendir(SERVER + filename)
		defer dh.Closedir()
		if err != nil {
			fmt.Fprintf(w, "failed to opendir %s (%s)", filename, err)
			return
		}
		list_dir(w, client, dh, r.URL.Path)
	} else {
		f, err := client.Open(SERVER+filename, 0, 0)
		if err != nil {
			fmt.Fprintf(w, "Failed to open %s (%s)", filename, err)
			return
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
	flag.StringVar(&SERVER, "server", "smb://localhost",
		"The smb server to use")
	flag.StringVar(&DOMAIN, "domain", "URT",
		"The domain to use")
	flag.Parse()
	log.Print(fmt.Sprintf("Using server %s (domain %s)\n", SERVER, DOMAIN))

	http.HandleFunc("/login", handler_login)
	http.HandleFunc("/logout", handler_logout)
	http.HandleFunc("/dfs/", handler_dfs)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	address := ":8080"
	log.Print("listen on ", address)
	http.ListenAndServe(address, nil)
}
