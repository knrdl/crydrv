package main

import (
	"log"
	"net/http"
	"strings"
	"time"
)

type AuthData struct {
	userKey  UserKey
	userSalt UserSalt
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
	if ok && strings.Count(username, "") > 0 && strings.Count(password, "") >= int(app.minPasswordLength) {

		handleRegistration := func(auth *AuthData) bool {
			if !app.openRegistration {
				userFingerprint := auth.userKey.hash(auth.userSalt)
				if !app.usersAllowlist.Contains(userFingerprint) {
					http.SetCookie(w, deleteCookie)
					log.Printf("user '%s' is not allowed to login with the provided password. Add '%s' to USERS_ALLOWLIST to grant permission.\n", username, strEncode(userFingerprint))
					http.Error(w, "unauthorized account", http.StatusForbidden)
					return false
				}
			}
			return true
		}

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

			if handleRegistration(auth) {
				return auth
			} else {
				return nil // handleRegistration has set the http response
			}
		} else {

			auth := new(AuthData)
			auth.userSalt = makeUserSalt(app.appKey, Username(username))
			auth.userKey = Password(password).hash(auth.userSalt)

			if handleRegistration(auth) {
				http.SetCookie(w, &http.Cookie{
					Name:     COOKIE_NAME,
					Value:    strEncode(auth.userKey),
					Expires:  time.Now().Add(app.cookieLifetime),
					Path:     "/",
					HttpOnly: true,
					SameSite: http.SameSiteLaxMode,
				})

				return auth
			} else {
				return nil // handleRegistration has set the http response
			}
		}
	} else {
		http.SetCookie(w, deleteCookie)
		w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return nil
	}
}
