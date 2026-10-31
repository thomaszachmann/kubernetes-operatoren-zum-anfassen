package controller

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
)

// fakeRegistrar spielt den Registrar im Testprozess: Map statt DNS.
type fakeRegistrar struct {
	*httptest.Server
	mu      sync.Mutex
	records map[string]bool
	down    atomic.Bool // true: jede Anfrage bekommt 503
}

func newFakeRegistrar() *fakeRegistrar {
	f := &fakeRegistrar{records: map[string]bool{}}
	f.Server = httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if f.down.Load() {
				http.Error(w, "Wartung", http.StatusServiceUnavailable)
				return
			}
			name := r.URL.Path[len("/records/"):]
			f.mu.Lock()
			defer f.mu.Unlock()
			switch r.Method {
			case http.MethodPut:
				f.records[name] = true
				w.WriteHeader(http.StatusOK)
			case http.MethodDelete:
				delete(f.records, name)
				w.WriteHeader(http.StatusNoContent)
			}
		}))
	return f
}

// has meldet, ob der Eintrag existiert.
func (f *fakeRegistrar) has(name string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.records[name]
}
