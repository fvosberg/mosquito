package mosquito

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	log "github.com/Sirupsen/logrus"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/pkg/errors"
)

// NewServer creates a new server for the todo service
// we are returning the interface, not the concrete type, to decouple our implementation from the API
func NewServer(authPubKey io.Reader) (http.Handler, error) {
	listHandler, err := authenticated(authPubKey, &httpListHandler{})
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

func (h *httpListHandler) ServeHTTP(customerID int, w http.ResponseWriter, r *http.Request) {
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

func authenticated(pub io.Reader, h authenticatedHandler) (http.Handler, error) {
	if pub == nil {
		return nil, errors.New("no pub key provided")
	}
	pubKeyData, err := ioutil.ReadAll(pub)
	if err != nil {
		return nil, errors.Wrap(err, "reading pub key failed")
	}
	pubKey, err := jwt.ParseRSAPublicKeyFromPEM(pubKeyData)
	if err != nil {
		return nil, errors.Wrap(err, "parsing pub key failed")
	}

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
		token, err := jwt.Parse(authHeader[7:], func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
				return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
			}
			return pubKey, nil
		})
		if err != nil {
			w.WriteHeader(401)
			fmt.Fprintf(w, `{"msg":"JWT could not be parsed correctly: %s"}`, err)
			return
		}
		if !token.Valid {
			w.WriteHeader(401)
			fmt.Fprint(w, `{"msg":"JWT invalid"}`)
			return
		}
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			w.WriteHeader(401)
			fmt.Fprint(w, `{"msg":"JWT claims invalid"}`)
			return
		}
		id, ok := claims["id"].(float64)
		if !ok {
			w.WriteHeader(400)
			fmt.Fprintf(w, `{"msg":"ID of type float in JWT missing, got %T"}`, claims["id"])
			return
		}
		h.ServeHTTP(int(id), w, r)
	}), nil
}

type authenticatedHandler interface {
	ServeHTTP(int, http.ResponseWriter, *http.Request)
}

type userHandler func(int, http.ResponseWriter, *http.Request)

func (f userHandler) ServeHTTP(ID int, w http.ResponseWriter, r *http.Request) {
	f(ID, w, r)
}
