package main

import (
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"time"
)

type AppData struct {
	appKey           AppKey
	openRegistration bool
	webBaseDir       string
}

func (app *AppData) handleRequest(w http.ResponseWriter, r *http.Request) {
	auth := app.handleAuth(w, r)
	if auth == nil {
		return
	}

	cryPath := path.Clean(r.URL.Path)
	if strings.HasSuffix(r.URL.Path, "/") {
		cryPath = path.Join(cryPath, "index.html")
	}

	filename := CryPath(cryPath).hash(auth.userKey, auth.userSalt)
	filepath := path.Join(string(auth.userDirPath), string(filename))

	switch r.Method {
	case "GET":
		if ok, err := IsFile(filepath); ok {
			if file, err := NewCryFileReader(FsPath(filepath), auth.userKey); err == nil {
				defer file.Close()
				http.ServeContent(w, r, cryPath, file.modTime, file) // cryPath for mime type detection by extension
			} else {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		} else if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return

		} else {
			http.Error(w, "not found", http.StatusNotFound)
			return

		}
	case "POST", "PUT":

		r.ParseMultipartForm(32 << 20) // read first 32MB into memory and spool to disk on overflow

		start := time.Now()

		file, handler, err := r.FormFile("file")
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		defer file.Close()

		WriteCryFile(FsPath(filepath), file, handler.Size, auth.userKey)

		if r.Method == "POST" {
			w.WriteHeader(http.StatusCreated)
			w.Header().Set("Location", r.URL.Path)
		} else {
			w.WriteHeader(http.StatusNoContent)
		}

		elapsed := time.Since(start)
		log.Printf("Upload took %s", elapsed.Round(time.Millisecond))
		return

	case "DELETE":
		if ok, err := IsFile(filepath); ok {
			if err = os.Remove(filepath); err == nil {
				w.WriteHeader(http.StatusNoContent)
				return
			} else {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		} else if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		} else {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
}

func main() {

	if os.Getenv("SECRET_KEY") == "" {
		log.Fatalf("Missing env var SECRET_KEY ... here is a good one: SECRET_KEY=%s", strEncode(Try(makeAppKey())))
	}

	app := new(AppData)
	app.appKey = Try(strDecode(os.Getenv("SECRET_KEY")))
	app.openRegistration = os.Getenv("OPEN_REGISTRATION") == "true"
	app.webBaseDir = "./www"
	Check(os.MkdirAll(app.webBaseDir, 0700))

	if app.openRegistration {
		log.Println("OPEN_REGISTRATION is enabled (every username/password combination can login)")
	} else {
		log.Println("OPEN_REGISTRATION is disabled (only existing users can login). Set env var OPEN_REGISTRATION=true to enable")
	}

	http.HandleFunc("/", app.handleRequest)
	log.Fatal(http.ListenAndServe(":8000", nil))
}
