package mosquito

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/pkg/errors"
)

// NewServer creates a new server for the todo service
// we are returning the interface, not the concrete type, to decouple our implementation from the API
func NewServer(jwtParser userIDer) (http.Handler, error) {
	listHandler, err := authenticated(jwtParser, &httpListHandler{})
	if err != nil {
		return nil, errors.Wrap(err, "creation of list handler failed")
	}
	return &server{
		listHandler: listHandler,
	}, nil
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

func (h *httpListHandler) ServeHTTP(userID string, w http.ResponseWriter, r *http.Request) {
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

//go:generate moq -out userider_moq.go . userIDer

type userIDer interface {
	UserID(string) (string, error)
}

func authenticated(tokenParser userIDer, h authenticatedHandler) (http.Handler, error) {
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
		id, err := tokenParser.UserID(authHeader[7:])
		if err != nil {
			// TODO forbidden / bad input
			w.WriteHeader(401)
			fmt.Fprintf(w, `{"msg":"%s"}`, err.Error())
			return
		}
		h.ServeHTTP(id, w, r)
	}), nil
}

type authenticatedHandler interface {
	ServeHTTP(string, http.ResponseWriter, *http.Request)
}

type userHandler func(string, http.ResponseWriter, *http.Request)

func (f userHandler) ServeHTTP(ID string, w http.ResponseWriter, r *http.Request) {
	f(ID, w, r)
}
