package main

import (
	"crypto/rand" 
	"path"
	"io"
	"fmt"
	"log"
	"html/template"
	"net/http"
	"os"
	"os/exec"
)

var GVFS_FUSE_ROOT = fmt.Sprintf("/run/user/%v/gvfs/", os.Getuid())

func mountDfs(username, password string) bool {
	domain := "URT"
	host := "naf1"
	smb_share := fmt.Sprintf("smb://%s@%s/%s", username, host, username)
	log.Print(smb_share)

	cmd := exec.Command("gvfs-mount", smb_share)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Fatal(err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		log.Fatal(err)
	}

	err = cmd.Start()
	if err != nil {
		log.Fatal(err)
	}
	stdin.Write([]byte(domain + "\n"))
	stdin.Write([]byte(password + "\n"))

	go io.Copy(os.Stdout, stdout) 
        go io.Copy(os.Stderr, stderr) 

	cmd.Wait()

	return true;
}

func getRandomString(length int) string {
    const alphanum = "0123456789abcdefghijklmnopqrstuvwxyz"
    var bytes = make([]byte, length)
    rand.Read(bytes)
    for i, b := range bytes {
        bytes[i] = alphanum[b % byte(len(alphanum))]
    }
    return string(bytes)
}

func createSymlink(username string) (target string, err error) {
	server := "naf1"

	fuse_dir := fmt.Sprintf(
		"%s/smb-share:server=%s,share=%s,user=%s",
		GVFS_FUSE_ROOT, server, username, username)
	target = fmt.Sprintf("dfs/%s", getRandomString(64))
	
	log.Print(fuse_dir)
	log.Print(target)

	err = os.Symlink(fuse_dir, target)
	if err != nil {
		log.Fatal(err)
	}
	return target, nil
}

func handler_login(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		username := r.FormValue("username")
		password := r.FormValue("password")

		if username == "" || password == "" {
			fmt.Fprintf(w, "Need username and password\n")
			return
		}

		// FIXME: CSRF protection (?)
		res := mountDfs(username, password)

		target, err := createSymlink(username)
		if err != nil {
			log.Fatal(err)
		}

		type LoginResult struct {
			Success bool
			Target string
		}
		r := LoginResult{res, target}
		t, _ := template.ParseFiles("login_done.html")
		t.Execute(w, r)

		return
	} 
	t, _ := template.ParseFiles("login.html")
	t.Execute(w, nil)
}

func handler_logout(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "logout")
}

func handler_dfs(w http.ResponseWriter, r *http.Request) {
	filename := path.Clean(r.URL.Path[1:])
	log.Print(filename)
	if filename == "dfs" {
		fmt.Fprintf(w, "dfs")
		return
	}
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		fmt.Fprintf(w, "404")
		return
	}
	h := http.StripPrefix("/dfs/", http.FileServer(http.Dir("./dfs/")))
	h.ServeHTTP(w, r)
}

// not reliable ?
func startGvfsFuse() {
	os.Mkdir("gvfs-fuse-mount2", 0750)
	cmd := exec.Command("/usr/lib/gvfs/gvfsd-fuse", "-f",  "gvfs-fuse-mount2/")
	cmd.Run()
}

func main() {
	// not reliable ?
	//go startGvfsFuse()

	http.HandleFunc("/login", handler_login)
	http.HandleFunc("/logout", handler_logout)
	http.HandleFunc("/dfs/", handler_dfs)

	address := ":8080"
	log.Print("listen on ", address)
	http.ListenAndServe(address, nil)
}
