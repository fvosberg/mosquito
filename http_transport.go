package mosquito

import "net/http"

// NewServer creates a new server for the todo service
// we are returning the interface, not the concrete type, to decouple our implementation from the API
func NewServer() http.Handler {
	return &server{}
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
