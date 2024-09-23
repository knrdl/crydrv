package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNoAuth(t *testing.T) {

	t.Setenv("SECRET_KEY", "q38rcNPrHkxBonj16HTKG95zbq0bzmJ189C9A-EgTxg")
	t.Setenv("OPEN_REGISTRATION", "true")

	app := makeAppData()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	if app.handleAuth(w, r) != nil {
		t.Error("handleAuth should return nil as a http response has been written")
	}
	res := w.Result()
	if res.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected unauthorized, got %v", res.StatusCode)
	}
}

func TestClosedRegistrationUserNotAllowed(t *testing.T) {

	t.Setenv("SECRET_KEY", "q38rcNPrHkxBonj16HTKG95zbq0bzmJ189C9A-EgTxg")

	app := makeAppData()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.SetBasicAuth("user1", "passwordpassword")
	w := httptest.NewRecorder()
	if app.handleAuth(w, r) != nil {
		t.Error("handleAuth should return nil as a http response has been written")
	}
	res := w.Result()
	if res.StatusCode != http.StatusForbidden {
		t.Errorf("expected forbidden, got %v", res.StatusCode)
	}
}

func TestClosedRegistrationUserAllowed(t *testing.T) {

	t.Setenv("SECRET_KEY", "q38rcNPrHkxBonj16HTKG95zbq0bzmJ189C9A-EgTxg")
	t.Setenv("USERS_ALLOWLIST", "jOoFNFNR1zZRWylgYRWi3PYnrn65Yc7AaAwUPTy9NyI")

	app := makeAppData()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.SetBasicAuth("user1", "passwordpassword")
	w := httptest.NewRecorder()
	if app.handleAuth(w, r) == nil {
		t.Error("handleAuth should not return nil as no http response has been written")
	}
	res := w.Result()
	if res.StatusCode != http.StatusOK {
		t.Errorf("expected OK, got %v", res.StatusCode)
	}
}

func TestCookieAuth(t *testing.T) {

	t.Setenv("SECRET_KEY", "q38rcNPrHkxBonj16HTKG95zbq0bzmJ189C9A-EgTxg")
	t.Setenv("OPEN_REGISTRATION", "true")

	app := makeAppData()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.SetBasicAuth("user1", "passwordpassword")
	w := httptest.NewRecorder()
	if app.handleAuth(w, r) == nil {
		t.Error("handleAuth should not return nil as no http response has been written")
	}
	res := w.Result()
	cookie := res.Cookies()[0]

	r = httptest.NewRequest(http.MethodGet, "/", nil)
	r.AddCookie(cookie)
	r.SetBasicAuth("user1", "passwordpassword")
	w = httptest.NewRecorder()
	if app.handleAuth(w, r) == nil {
		t.Error("handleAuth should not return nil as no http response has been written")
	}

}
