package mosquito

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
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

func TestHTTPListHandler(t *testing.T) {
	tests := map[string]struct {
		req                    *http.Request
		listerReturn           []Todo
		listerError            error
		expectedStatusCode     int
		expectedResponseHeader http.Header
		expectedResponseBody   string
	}{
		"happy": {
			req: httptest.NewRequest("GET", "/", nil),
			listerReturn: []Todo{
				{ID: "ONE", Title: "Test one", Author: "USER-ONE",
					CreatedAt: time.Date(2017, time.August, 1, 15, 45, 0, 0, time.UTC)},
				{ID: "TWO", Title: "Test two", Author: "USER-TWO",
					CreatedAt: time.Date(2017, time.August, 1, 15, 46, 0, 0, time.UTC)},
			},
			listerError:        nil,
			expectedStatusCode: 200,
			expectedResponseHeader: http.Header{
				"Content-Type": []string{"application/json; charset=UTF-8"},
			},
			expectedResponseBody: `[{"id":"ONE","title":"Test one","author":"USER-ONE","created_at":"2017-08-01T15:45:00Z","due_date":null},{"id":"TWO","title":"Test two","author":"USER-TWO","created_at":"2017-08-01T15:46:00Z","due_date":null}]`,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			listerMock := &listerMock{
				ListFunc: func() ([]Todo, error) {
					return tt.listerReturn, tt.listerError
				},
			}
			h := &httpListHandler{
				lister: listerMock,
			}
			res := httptest.NewRecorder()
			h.ServeHTTP(res, tt.req)

			if res.Code != tt.expectedStatusCode {
				t.Errorf("Expected status code %d, but got %d", tt.expectedStatusCode, res.Code)
			}
			if !cmp.Equal(res.HeaderMap, tt.expectedResponseHeader) {
				t.Fatalf("Unexpected header\nexpected: %#v\nactual:   %#v",
					tt.expectedResponseHeader,
					res.HeaderMap,
				)
			}
			if strings.Trim(res.Body.String(), "\n") != tt.expectedResponseBody {
				t.Errorf("Unexpected response body\nexpected: %s\nactual:   %s",
					tt.expectedResponseBody, res.Body.String())
			}
		})
	}
}
