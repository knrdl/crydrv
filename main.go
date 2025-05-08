package main

import (
	"errors"
	"log"
	"math"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

type AppData struct {
	appKey            AppKey
	openRegistration  bool
	usersAllowlist    UserFingerprints
	webBaseDir        string
	minPasswordLength uint32
	cookieLifetime    time.Duration
}

func (app *AppData) handleRequest(w http.ResponseWriter, r *http.Request) {
	urlPath := path.Clean(r.URL.Path)
	if len(urlPath) == 0 {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	if strings.HasSuffix(r.URL.Path, "/") {
		urlPath = path.Join(urlPath, "index.html")
	}

	auth := app.handleAuth(w, r)
	if auth == nil {
		// handleAuth has already set the http response
		return
	}

	cryName := CryPath(urlPath).hash(auth.userKey, auth.userSalt)
	fsPath := cryName.toFilepath(app.webBaseDir)

	switch r.Method {
	case "GET", "HEAD":
		fsPath.ReadLock()
		defer fsPath.ReadUnlock()
		if file, err := NewCryFileReader(fsPath, auth.userKey); err == nil {
			defer CheckFunc(file.Close)
			http.ServeContent(w, r, urlPath, file.modTime, file) // urlPath for mime type detection by extension
		} else if errors.Is(err, os.ErrNotExist) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		} else {
			http.Error(w, sanitizeError(err), http.StatusBadRequest)
			return
		}

	case "POST", "PUT":
		if err := r.ParseMultipartForm(32 << 20); err != nil { // read first 32MiB into memory and spool to disk on overflow
			http.Error(w, sanitizeError(err), http.StatusBadRequest)
			return
		}

		file, handler, err := r.FormFile("file")
		if err != nil {
			http.Error(w, sanitizeError(err), http.StatusBadRequest)
			return
		}
		defer CheckFunc(file.Close)

		if err := WriteCryFile(fsPath, file, handler.Size, auth.userKey); err != nil {
			http.Error(w, sanitizeError(err), http.StatusInternalServerError)
			return
		}

		if r.Method == "POST" {
			w.Header().Set("Location", r.URL.Path)
			w.WriteHeader(http.StatusCreated)
		} else {
			w.WriteHeader(http.StatusNoContent)
		}

		return

	case "DELETE":
		if ok, err := IsFile(string(fsPath)); err == nil && ok {
			if err = os.Remove(string(fsPath)); err == nil {
				w.WriteHeader(http.StatusNoContent)
				return
			} else {
				http.Error(w, sanitizeError(err), http.StatusInternalServerError)
				return
			}
		} else if err != nil {
			http.Error(w, sanitizeError(err), http.StatusInternalServerError)
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

func addSecurityHeaders(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Referrer-Policy", "strict-origin")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "sameorigin")
		next(w, r)
	}
}

func makeAppData() (app AppData) {
	if os.Getenv("SECRET_KEY") == "" {
		log.Fatalf("Missing env var SECRET_KEY ... here is a good one: SECRET_KEY=%s", strEncode(Try(makeAppKey())))
	}
	app.appKey = Try(strDecode(os.Getenv("SECRET_KEY")))
	if len(app.appKey) != APP_KEY_LENGTH {
		log.Fatal("Wrong length for env var SECRET_KEY")
	}

	app.openRegistration = os.Getenv("OPEN_REGISTRATION") == "true"
	if app.openRegistration {
		log.Println("OPEN_REGISTRATION is enabled (every username/password combination can login)")
	} else {
		log.Println("OPEN_REGISTRATION is disabled (only users on allowlist can login). Set env var OPEN_REGISTRATION=true to enable")
		if err := app.usersAllowlist.Load(os.Getenv("USERS_ALLOWLIST")); err != nil {
			log.Fatal("USERS_ALLOWLIST contains invalid values:", err.Error())
		}
		if len(app.usersAllowlist) == 0 {
			log.Println("USERS_ALLOWLIST contains no values. Nobody can login")
		} else {
			log.Println("USERS_ALLOWLIST contains", len(app.usersAllowlist), "records")
		}
	}

	app.minPasswordLength = 16
	if minPasswordLengthStr := os.Getenv("MIN_PASSWORD_LENGTH"); minPasswordLengthStr != "" {
		minPasswordLength, err := strconv.ParseUint(minPasswordLengthStr, 10, 32)
		if err == nil && minPasswordLength > 0 && minPasswordLength <= math.MaxUint32 {
			app.minPasswordLength = uint32(minPasswordLength)
		} else {
			log.Fatalf("invalid value for MIN_PASSWORD_LENGTH provided")
		}
	}
	log.Println("MIN_PASSWORD_LENGTH is set to", app.minPasswordLength)

	app.cookieLifetime = 24 * time.Hour

	app.webBaseDir = "./www"
	Check(os.MkdirAll(app.webBaseDir, 0700))

	return app
}

func main() {
	app := makeAppData()

	http.HandleFunc("/", addSecurityHeaders(app.handleRequest))
	log.Fatal(http.ListenAndServe(":8000", nil))
}
