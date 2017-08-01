package mosquito

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPRouting(t *testing.T) {
	teapot := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(413)
	})

	tests := map[string]struct {
		set                func(*server)
		req                *http.Request
		expectedStatusCode int
	}{
		"404": {
			req:                httptest.NewRequest("GET", "/foobar", nil),
			expectedStatusCode: 404,
		},
		"list": {
			req:                httptest.NewRequest("GET", "/", nil),
			expectedStatusCode: 413,
			set: func(s *server) {
				s.listHandler = teapot
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			srv := NewServer()
			if tt.set != nil {
				i := srv.(*server)
				tt.set(i)
			}
			res := httptest.NewRecorder()
			srv.ServeHTTP(res, tt.req)
			if res.Code != tt.expectedStatusCode {
				t.Fatalf("Expected status code %d, got %d", tt.expectedStatusCode, res.Code)
			}
		})
	}
}
