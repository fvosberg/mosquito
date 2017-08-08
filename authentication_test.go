package mosquito

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/pkg/errors"
)

func TestJWTAuth(t *testing.T) {
	inOneMinute := time.Now().Add(time.Minute)
	tests := map[string]struct {
		jwtID string
	}{
		"valid": {
			jwtID: "1337",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			pubKey, err := os.Open(filepath.Join("testdata", "public.pem"))
			if err != nil {
				t.Fatalf("Could not open testdata public pem: %s", err)
			}
			auth, err := newJWTAuth(pubKey)
			if err != nil {
				t.Fatalf("Could not create auth: %s", err)
			}
			testJWT, err := newJWT(filepath.Join("testdata", "private.pem"), inOneMinute, tt.jwtID)
			if err != nil {
				t.Fatalf("Could not create test JWT: %s", err)
			}
			id, err := auth.UserID(testJWT)
			if err != nil {
				t.Fatalf("Unexpected error on token parsing: %s", err)
			}
			if id != tt.jwtID {
				t.Fatalf(`Unexpected id "%s", expected "%s"`, id, tt.jwtID)
			}
		})
	}
}

func newJWT(privKeyPath string, exp time.Time, id string) (string, error) {
	data, err := ioutil.ReadFile(privKeyPath)
	if err != nil {
		return "", errors.Wrap(err, "priv key loading failed")
	}
	privKey, err := jwt.ParseRSAPrivateKeyFromPEM(data)
	if err != nil {
		return "", errors.Wrapf(err, `priv key "%s" parsing failed`, privKeyPath)
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS512, jwt.MapClaims{
		"exp": exp.Unix(),
		"id":  id,
	})
	return token.SignedString(privKey)
}
