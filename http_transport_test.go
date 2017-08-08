package mosquito

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
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
			srv, err := NewServer(&userIDerMock{})
			if err != nil {
				t.Fatalf("Error on creation of server: %s", err)
			}
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
			req: func() *http.Request {
				r := httptest.NewRequest("GET", "/", nil)
				r.Header.Set("Authentication", "Bearer JWT")
				return r
			}(),
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
		"error on list retrieving": {
			req: func() *http.Request {
				r := httptest.NewRequest("GET", "/", nil)
				r.Header.Set("Authentication", "Bearer JWT")
				return r
			}(),
			listerReturn:       nil,
			listerError:        errors.New("something went wrong :("),
			expectedStatusCode: 500,
			expectedResponseHeader: http.Header{
				"Content-Type": []string{"application/json; charset=UTF-8"},
			},
			expectedResponseBody: `{"msg":"Internal Server Error"}`,
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
			h.ServeHTTP("", res, tt.req)

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

func validAuthPubKey(t *testing.T) io.Reader {
	r, err := os.Open(filepath.Join("testdata", "public.pem"))
	if err != nil {
		t.Fatalf("Could not open auth pub key: %s", err)
	}
	return r
}

func TestAuthenticated(t *testing.T) {
	tests := map[string]struct {
		authPubKey                 io.Reader
		req                        *http.Request
		expectedJWT                string
		expectedUserIDInSubHandler string
		expectedStatusCode         int
		expectedResponseHeader     http.Header
		expectedResponseBody       string
	}{
		"happy": {
			authPubKey: validAuthPubKey(t),
			req: func() *http.Request {
				r := httptest.NewRequest("GET", "/", nil)
				r.Header.Set("Authentication", "Bearer USERJWT")
				return r
			}(),
			expectedJWT:                "USERJWT",
			expectedUserIDInSubHandler: "1337",
			expectedStatusCode:         413,
			expectedResponseHeader: http.Header{
				"Content-Type": []string{"application/json; charset=UTF-8"},
			},
			expectedResponseBody: `"called inner handler"`,
		},
		"missing authentication header": {
			authPubKey:         validAuthPubKey(t),
			req:                httptest.NewRequest("GET", "/", nil),
			expectedStatusCode: 400,
			expectedResponseHeader: http.Header{
				"Content-Type": []string{"application/json; charset=UTF-8"},
			},
			expectedResponseBody: `{"msg":"Missing \"Authentication\" header of format \"Bearer [JWT]\""}`,
		},
		"wrongly formatted auth header": {
			authPubKey: validAuthPubKey(t),
			req: func() *http.Request {
				r := httptest.NewRequest("GET", "/", nil)
				r.Header.Set("Authentication", "JWT")
				return r
			}(),
			expectedStatusCode: 400,
			expectedResponseHeader: http.Header{
				"Content-Type": []string{"application/json; charset=UTF-8"},
			},
			expectedResponseBody: `{"msg":"Wrongly formatted \"Authentication\" header. It must be of the format \"Bearer [JWT]\""}`,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			tokenParser := &userIDerMock{
				UserIDFunc: func(token string) (string, error) {
					return tt.expectedUserIDInSubHandler, nil
				},
			}
			userIDInSubHandler := ""
			h := userHandler(func(ID string, w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(413)
				w.Write([]byte(`"called inner handler"`))
				userIDInSubHandler = ID
			})

			res := httptest.NewRecorder()
			ah, err := authenticated(tokenParser, h)
			if err != nil {
				t.Fatalf("Error on creating authentication wrapper: %s", err)
			}
			ah.ServeHTTP(res, tt.req)

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
			if userIDInSubHandler != tt.expectedUserIDInSubHandler {
				t.Errorf("Expecting user ID %s, but got %s", tt.expectedUserIDInSubHandler, userIDInSubHandler)
			}
		})
	}
}
