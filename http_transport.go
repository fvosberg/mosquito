package mosquito

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	log "github.com/Sirupsen/logrus"
)

// NewServer creates a new server for the todo service
// we are returning the interface, not the concrete type, to decouple our implementation from the API
func NewServer() http.Handler {
	return &server{
		listHandler: authenticated(&httpListHandler{}),
	}
}

type server struct {
	mux *http.ServeMux

	listHandler http.Handler
}

// ServeHTTP implements the http.Handler
func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if s.mux == nil {
		s.initServeMux()
	}
	s.mux.ServeHTTP(w, r)
}

// initServeMux initializes the internal ServeMux
// This function is separated, for testing. This offers the ability to test the routing encapsulated
func (s *server) initServeMux() {
	s.mux = http.NewServeMux()
	s.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		s.listHandler.ServeHTTP(w, r)
	})
}

type httpListHandler struct {
	lister lister
}

func (h *httpListHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	todos, err := h.lister.List()
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprint(w, `{"msg":"Internal Server Error"}`)
		log.Errorf("Retrieving todo list failed: %s", err)
		return
	}
	json.NewEncoder(w).Encode(todos)
}

//go:generate moq -out lister_moq.go . lister

type lister interface {
	List() ([]Todo, error)
}

func authenticated(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		authHeader := r.Header.Get("Authentication")
		if authHeader == "" {
			w.WriteHeader(400)
			fmt.Fprintf(w, `{"msg":"Missing \"Authentication\" header of format \"Bearer [JWT]\""}`)
			return
		}
		if !strings.HasPrefix(authHeader, "Bearer ") {
			w.WriteHeader(400)
			fmt.Fprintf(w, `{"msg":"Wrongly formatted \"Authentication\" header. It must be of the format \"Bearer [JWT]\""}`)
			return
		}
		h.ServeHTTP(w, r)
	})
}

type authenticatedHandler interface {
	ServeHTTP(int, http.ResponseWriter, *http.Request)
}
