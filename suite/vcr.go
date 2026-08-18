package suite

import (
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"path/filepath"
	"runtime"
	"testing"

	"gopkg.in/dnaeon/go-vcr.v4/pkg/cassette"
	"gopkg.in/dnaeon/go-vcr.v4/pkg/recorder"
)

func parsePath(urlString string) string {
	u, err := url.Parse(urlString)

	if err != nil {
		panic(err)
	}

	return u.Path
}

type tHelper interface {
	Fatal(args ...any)
	Error(args ...any)
	Cleanup(func())
}

func UseCassette(t tHelper, name string) *http.Client {
	_, filename, _, _ := runtime.Caller(0)

	projectRoot := filepath.Dir(filepath.Dir(filename))

	opts := []recorder.Option{
		recorder.WithMatcher(func(r1 *http.Request, r2 cassette.Request) bool {

			return r1.Method == r2.Method && parsePath(r1.URL.String()) == parsePath(r2.URL)
		}),
		recorder.WithHook(func(i *cassette.Interaction) error {
			if i.Request.Headers.Get("Authorization") != "" {
				i.Request.Headers.Set("Authorization", "***********************")
			}

			if i.Request.Headers.Get("x-api-key") != "" {
				i.Request.Headers.Set("x-api-key", "***********************")
			}

			return nil
		}, recorder.BeforeSaveHook),
	}

	r, err := recorder.New(
		filepath.Join(projectRoot, "suite", "cassettes", name),
		opts...,
	)

	if r.Mode() != recorder.ModeRecordOnce {
		t.Fatal("Recorder should be in ModeRecordOnce")
	}

	if err != nil {
		panic(err)
	}

	t.Cleanup(func() {
		if err := r.Stop(); err != nil {
			t.Error(err)
		}
	})

	c := r.GetDefaultClient()
	j, _ := cookiejar.New(nil)

	c.Jar = j

	return c
}

type ManualCassette struct {
	httpClient *http.Client
	recorder   *recorder.Recorder
	t          *testing.T
}

func (m *ManualCassette) HttpClient() *http.Client {
	return m.httpClient
}

func (m *ManualCassette) Stop() {
	if err := m.recorder.Stop(); err != nil {
		m.t.Error(err)
	}
}

func UseManualCassette(t *testing.T, name string) *ManualCassette {
	_, filename, _, _ := runtime.Caller(0)

	projectRoot := filepath.Dir(filepath.Dir(filename))

	opts := []recorder.Option{
		recorder.WithMatcher(func(r1 *http.Request, r2 cassette.Request) bool {
			return r1.Method == r2.Method && parsePath(r1.URL.String()) == parsePath(r2.URL)
		}),
		recorder.WithHook(func(i *cassette.Interaction) error {
			if i.Request.Headers.Get("Authorization") != "" {
				i.Request.Headers.Set("Authorization", "***********************")
			}

			if i.Request.Headers.Get("x-api-key") != "" {
				i.Request.Headers.Set("x-api-key", "*****************")
			}

			return nil
		}, recorder.BeforeSaveHook),
	}

	r, err := recorder.New(
		filepath.Join(projectRoot, "suite", "cassettes", name),
		opts...,
	)

	if r.Mode() != recorder.ModeRecordOnce {
		t.Fatal("Recorder should be in ModeRecordOnce")
	}

	if err != nil {
		panic(err)
	}

	return &ManualCassette{
		recorder:   r,
		httpClient: r.GetDefaultClient(),
	}
}
