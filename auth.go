package main

import (
	"net/http"
	"os"
	"path"
	"strings"
	"time"
)

type AuthData struct {
	userKey     UserKey
	userSalt    UserSalt
	userDirPath FsPath
}

const COOKIE_NAME = "crydrv"

var deleteCookie = &http.Cookie{
	Name:     COOKIE_NAME,
	Value:    "",
	Path:     "/",
	Expires:  time.Unix(0, 0),
	HttpOnly: true,
}

func (app *AppData) handleAuth(w http.ResponseWriter, r *http.Request) *AuthData {
	username, password, ok := r.BasicAuth()
	if ok && strings.Count(username, "") > 0 && strings.Count(password, "") > 16 {
		if cookie, err := r.Cookie(COOKIE_NAME); err == nil {
			userKey, err := strDecode(cookie.Value)
			if err != nil {
				http.SetCookie(w, deleteCookie)
				http.Error(w, sanitizeError(err), http.StatusBadRequest)
				return nil
			}
			if len(userKey) != USER_KEY_LENGTH {
				http.SetCookie(w, deleteCookie)
				http.Error(w, "invalid key in cookie", http.StatusBadRequest)
				return nil
			}
			auth := new(AuthData)
			auth.userSalt = makeUserSalt(app.appKey, Username(username))
			auth.userKey = userKey
			userDir := strEncode(auth.userKey.hash(auth.userSalt))
			auth.userDirPath = FsPath(path.Join(app.webBaseDir, userDir))

			if ok, err := IsDir(string(auth.userDirPath)); ok && err == nil {
				return auth
			} else if err != nil {
				http.SetCookie(w, deleteCookie)
				http.Error(w, sanitizeError(err), http.StatusInternalServerError)
				return nil
			} else {
				http.SetCookie(w, deleteCookie)
				http.Error(w, "wrong key, please login again", http.StatusBadRequest)
				return nil
			}

		} else {
			auth := new(AuthData)
			auth.userSalt = makeUserSalt(app.appKey, Username(username))
			auth.userKey = Password(password).hash(auth.userSalt)
			userDir := strEncode(auth.userKey.hash(auth.userSalt))
			auth.userDirPath = FsPath(path.Join(app.webBaseDir, userDir))

			if app.openRegistration {
				if err := os.MkdirAll(string(auth.userDirPath), 0700); err != nil {
					http.Error(w, sanitizeError(err), http.StatusInternalServerError)
					return nil
				}
			} else {
				ok, err := IsDir(string(auth.userDirPath))
				if err != nil {
					http.Error(w, sanitizeError(err), http.StatusInternalServerError)
					return nil
				} else if !ok {
					http.Error(w, "unknown account", http.StatusBadRequest)
					return nil
				}
			}

			http.SetCookie(w, &http.Cookie{
				Name:     COOKIE_NAME,
				Value:    strEncode(auth.userKey),
				Expires:  time.Now().Add(5 * time.Hour),
				Path:     "/",
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode,
			})

			return auth
		}
	} else {
		http.SetCookie(w, deleteCookie)
		w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return nil
	}
}
