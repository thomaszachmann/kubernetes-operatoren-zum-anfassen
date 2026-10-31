// registrar ist ein Lab-"DNS-Registrar": ein externer Dienst, den der
// site-operator (Tag 7) per HTTP pflegt. Daten liegen nur im Speicher.
//
//	PUT    /records/<name>  {"target":"..."}  -> 201 neu, 200 geändert
//	GET    /records                           -> 200 alle Einträge
//	DELETE /records/<name>                    -> 204 oder 404
//	PUT    /admin/outage   / DELETE /admin/outage -> Ausfall an/aus (503)
package main

import (
	"encoding/json"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"regexp"
	"sort"
	"sync"
)

// Record ist ein DNS-Eintrag: Name -> Ziel (z. B. Service-URL).
type Record struct {
	Name   string `json:"name"`
	Target string `json:"target"`
}

type registrar struct {
	mu      sync.Mutex
	records map[string]Record
	outage  bool
}

var validName = regexp.MustCompile(`^[a-z0-9]([-a-z0-9.]{0,251}[a-z0-9])?$`)

func main() {
	addr := flag.String("addr", "127.0.0.1:18090", "Listen-Adresse")
	flag.Parse()
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	r := &registrar{records: map[string]Record{}}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /records", r.list)
	mux.HandleFunc("PUT /records/{name}", r.put)
	mux.HandleFunc("DELETE /records/{name}", r.del)
	mux.HandleFunc("PUT /admin/outage", r.setOutage(true))
	mux.HandleFunc("DELETE /admin/outage", r.setOutage(false))

	log.Info("registrar startet", "addr", *addr)
	h := logging(log, r.guard(mux))
	if err := http.ListenAndServe(*addr, h); err != nil {
		log.Error("beendet", "err", err)
		os.Exit(1)
	}
}

// guard simuliert einen Ausfall: Alles außer /admin/ liefert dann 503.
func (r *registrar) guard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		r.mu.Lock()
		down := r.outage
		r.mu.Unlock()
		if down && req.URL.Path != "/admin/outage" {
			http.Error(w, "registrar: Wartung", http.StatusServiceUnavailable)
			return
		}
		next.ServeHTTP(w, req)
	})
}

func (r *registrar) list(w http.ResponseWriter, _ *http.Request) {
	r.mu.Lock()
	out := make([]Record, 0, len(r.records))
	for _, rec := range r.records {
		out = append(out, rec)
	}
	r.mu.Unlock()
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	writeJSON(w, http.StatusOK, out)
}

func (r *registrar) put(w http.ResponseWriter, req *http.Request) {
	name := req.PathValue("name")
	var body struct {
		Target string `json:"target"`
	}
	err := json.NewDecoder(http.MaxBytesReader(w, req.Body, 4096)).Decode(&body)
	if err != nil || body.Target == "" || !validName.MatchString(name) {
		http.Error(w, "ungültiger Name oder target fehlt", http.StatusBadRequest)
		return
	}
	r.mu.Lock()
	_, existed := r.records[name]
	rec := Record{Name: name, Target: body.Target}
	r.records[name] = rec
	r.mu.Unlock()
	status := http.StatusCreated
	if existed {
		status = http.StatusOK // PUT ist idempotent
	}
	writeJSON(w, status, rec)
}

func (r *registrar) del(w http.ResponseWriter, req *http.Request) {
	name := req.PathValue("name")
	r.mu.Lock()
	_, ok := r.records[name]
	delete(r.records, name)
	r.mu.Unlock()
	if !ok {
		http.Error(w, "nicht gefunden", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (r *registrar) setOutage(on bool) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		r.mu.Lock()
		r.outage = on
		r.mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func logging(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		log.Info("request", "method", req.Method, "path", req.URL.Path)
		next.ServeHTTP(w, req)
	})
}
