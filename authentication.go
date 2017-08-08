package mosquito

import (
	"crypto/rsa"
	"fmt"
	"io"
	"io/ioutil"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/fvosberg/errtypes"
	"github.com/pkg/errors"
)

func newJWTAuth(pub io.Reader) (*jwtAuth, error) {
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

	return &jwtAuth{
		pubKey: pubKey,
	}, nil
}

type jwtAuth struct {
	pubKey *rsa.PublicKey
}

func (a *jwtAuth) UserID(s string) (string, error) {
	token, err := jwt.Parse(s, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}
		return a.pubKey, nil
	})
	if err != nil {
		return "", errtypes.NewUnauthenticatedErrorf("JWT could not be parsed correctly: %s", err)
	}
	if !token.Valid {
		return "", errtypes.NewUnauthenticatedError("JWT invalid")
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", errtypes.NewUnauthenticatedError("JWT claims invalid")
	}
	id, ok := claims["id"].(string)
	if !ok {
		return "", errtypes.NewBadInputErrorf("ID of type string in JWT missing, got %T", claims["id"])
	}
	return id, nil
}
