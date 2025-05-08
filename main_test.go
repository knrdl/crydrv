package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	flag.Parse()
	returnCode := m.Run()

	if returnCode == 0 && testing.CoverMode() != "" {
		c := testing.Coverage()
		if c < 0.33 {
			fmt.Println("Tests passed but coverage failed at", c)
			returnCode = 1
		}
	}
	os.Exit(returnCode)
}

func TestAppDataParsing(t *testing.T) {
	t.Setenv("SECRET_KEY", "q38rcNPrHkxBonj16HTKG95zbq0bzmJ189C9A-EgTxg")
	t.Setenv("OPEN_REGISTRATION", "true")
	t.Setenv("MIN_PASSWORD_LENGTH", "123")
	t.Setenv("USERS_ALLOWLIST", "1,2,3")

	app := makeAppData()

	if !bytes.Equal(app.appKey, []byte{171, 127, 43, 112, 211, 235, 30, 76, 65, 162, 120, 245, 232, 116, 202, 27, 222, 115, 110, 173, 27, 206, 98, 117, 243, 208, 189, 3, 225, 32, 79, 24}) {
		t.Error("wrong appkey parsed")
	}

	if app.minPasswordLength != 123 {
		t.Error("wrong minPasswordLength parsed")
	}

	if app.usersAllowlist != nil {
		t.Error("wrong usersAllowlist parsed, should be nil as registration is open")
	}
}

func TestUnallowedMethod(t *testing.T) {
	t.Setenv("SECRET_KEY", "q38rcNPrHkxBonj16HTKG95zbq0bzmJ189C9A-EgTxg")
	t.Setenv("OPEN_REGISTRATION", "true")
	app := makeAppData()
	r := httptest.NewRequest(http.MethodPatch, "/", nil)
	r.SetBasicAuth("user1", "passwordpassword") // 16 chars
	w := httptest.NewRecorder()
	addSecurityHeaders(app.handleRequest)(w, r)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %v", w.Code)
	}
}

func TestCRUD(t *testing.T) {

	t.Setenv("SECRET_KEY", "q38rcNPrHkxBonj16HTKG95zbq0bzmJ189C9A-EgTxg")
	t.Setenv("OPEN_REGISTRATION", "true")
	app := makeAppData()

	app.webBaseDir = "./www-test"
	defer CheckFunc(func() error { return os.RemoveAll(app.webBaseDir) })
	Check(os.MkdirAll(app.webBaseDir, 0700))

	const fileContent1 = "test1"
	const fileContent2 = "test2"

	t.Run("it should not be found", func(t0 *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.SetBasicAuth("user1", "passwordpassword") // 16 chars
		w := httptest.NewRecorder()
		app.handleRequest(w, r)
		if w.Code != http.StatusNotFound {
			t0.Errorf("expected 404, got %v", w.Code)
		}
	})

	t.Run("it should be created", func(t0 *testing.T) {
		pr, pw := io.Pipe()
		//This writers is going to transform
		//what we pass to it to multipart form data
		//and write it to our io.Pipe
		writer := multipart.NewWriter(pw)

		go func() {
			defer CheckFunc(writer.Close)
			//we create the form data field 'fileupload'
			//wich returns another writer to write the actual file
			part, err := writer.CreateFormFile("file", "index.html")
			if err != nil {
				t0.Error(err)
			}

			_, err = part.Write([]byte(fileContent1))
			if err != nil {
				t0.Error(err)
			}
		}()

		r := httptest.NewRequest(http.MethodPost, "/", pr)
		r.Header.Add("Content-Type", writer.FormDataContentType())
		r.SetBasicAuth("user1", "passwordpassword")
		w := httptest.NewRecorder()
		app.handleRequest(w, r)
		if w.Code != http.StatusCreated {
			res := w.Result()
			defer CheckFunc(res.Body.Close)
			data := Try(io.ReadAll(res.Body))
			t0.Errorf("expected 201, got %+v %s", res, data)
		}
	})

	t.Run("it should be found", func(t0 *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r.SetBasicAuth("user1", "passwordpassword") // 16 chars
		w := httptest.NewRecorder()
		app.handleRequest(w, r)
		if w.Code != http.StatusOK {
			t0.Errorf("expected 200, got %v", w.Code)
		}
		res := w.Result()
		defer CheckFunc(res.Body.Close)
		data := Try(io.ReadAll(res.Body))
		if string(data) != fileContent1 {
			t0.Errorf("received wrong content: %v", data)
		}
	})

	t.Run("it should be updated", func(t0 *testing.T) {
		pr, pw := io.Pipe()
		//This writers is going to transform
		//what we pass to it to multipart form data
		//and write it to our io.Pipe
		writer := multipart.NewWriter(pw)

		go func() {
			defer CheckFunc(writer.Close)
			//we create the form data field 'fileupload'
			//wich returns another writer to write the actual file
			part, err := writer.CreateFormFile("file", "index.html")
			if err != nil {
				t0.Error(err)
			}

			_, err = part.Write([]byte(fileContent2))
			if err != nil {
				t0.Error(err)
			}
		}()

		r := httptest.NewRequest(http.MethodPut, "/", pr)
		r.Header.Add("Content-Type", writer.FormDataContentType())
		r.SetBasicAuth("user1", "passwordpassword")
		w := httptest.NewRecorder()
		app.handleRequest(w, r)
		if w.Code != http.StatusNoContent {
			res := w.Result()
			defer CheckFunc(res.Body.Close)
			data := Try(io.ReadAll(res.Body))
			t0.Errorf("expected 204, got %+v %s", res, data)
		}
	})

	t.Run("it should be deleted", func(t0 *testing.T) {
		r := httptest.NewRequest(http.MethodDelete, "/", nil)
		r.SetBasicAuth("user1", "passwordpassword") // 16 chars
		w := httptest.NewRecorder()
		app.handleRequest(w, r)
		if w.Code != http.StatusNoContent {
			t0.Errorf("expected 204, got %v", w.Code)
		}
	})

	t.Run("it should not be found again", func(t0 *testing.T) {
		r := httptest.NewRequest(http.MethodHead, "/", nil)
		r.SetBasicAuth("user1", "passwordpassword") // 16 chars
		w := httptest.NewRecorder()
		app.handleRequest(w, r)
		if w.Code != http.StatusNotFound {
			t0.Errorf("expected 404, got %v", w.Code)
		}
	})

}
