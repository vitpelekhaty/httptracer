package httptracer

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func echo(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")

	if strings.Trim(username, " ") == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("Hello, %s", username)))
}

func traceEcho(client *http.Client, uri, path string, options ...Option) error {
	empty := true

	f, err := os.Create(path)

	if err != nil {
		return err
	}

	defer func() {
		f.WriteString("]")
		f.Close()
	}()

	f.WriteString("[")

	callback := func(entry *Entry) {
		if !empty {
			f.WriteString(",")
		}

		empty = false
	}

	opts := []Option{WithWriter(f), WithCallback(callback)}

	if len(options) > 0 {
		opts = append(opts, options...)
	}

	client = Trace(client, opts...)

	form := url.Values{}
	form.Set("username", "John Doe")

	req, err := http.NewRequest("POST", uri, strings.NewReader(form.Encode()))

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded;charset=utf-8")

	if err != nil {
		return err
	}

	_, err = client.Do(req)

	return err
}

func TestTrace(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(echo))
	defer ts.Close()

	_, exec, _, ok := runtime.Caller(0)

	if !ok {
		t.FailNow()
	}

	path := filepath.Join(filepath.Dir(exec), "/testdata/trace")
	client := &http.Client{Timeout: time.Second * 5}

	err := traceEcho(client, ts.URL+"/echo", path, WithBodies(true))

	if err != nil {
		t.Fatal(err)
	}

	if fi, err := os.Stat(path); err == nil {
		if fi.Size() <= 4 {
			t.FailNow()
		}
	} else {
		t.Fatal(err)
	}
}
