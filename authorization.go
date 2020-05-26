package httpclient

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"time"

	jwt "github.com/gbrlsnchs/jwt"
	"github.com/pkg/errors"
	"gopkg.in/h2non/gentleman.v2"
	"gopkg.in/h2non/gentleman.v2/context"
	"gopkg.in/h2non/gentleman.v2/plugin"
)

// Token is JWT token.
// We need to define our own structure as the library define aud as a string but it can also be a string array.
// To fix this issue, we remove aud as we do not use it here.
type Token struct {
	hdr            *header
	Issuer         string `json:"iss,omitempty"`
	Subject        string `json:"sub,omitempty"`
	ExpirationTime int64  `json:"exp,omitempty"`
	NotBefore      int64  `json:"nbf,omitempty"`
	IssuedAt       int64  `json:"iat,omitempty"`
	ID             string `json:"jti,omitempty"`
	Username       string `json:"preferred_username,omitempty"`
}

type header struct {
	Algorithm   string `json:"alg,omitempty"`
	KeyID       string `json:"kid,omitempty"`
	Type        string `json:"typ,omitempty"`
	ContentType string `json:"cty,omitempty"`
}

// NewBasicAuthClient creates a new HTTP client using a basic authentication
func NewBasicAuthClient(addrAPI string, reqTimeout time.Duration, username, password string) (*Client, error) {
	var token = base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
	return New(addrAPI, reqTimeout, func(r *gentleman.Request) (*gentleman.Request, error) {
		return r.SetHeader("Authorization", "Basic "+token), nil
	})
}

// NewBearerAuthClient creates a new HTTP client using a bearer authentication
func NewBearerAuthClient(addrAPI string, reqTimeout time.Duration, tokenProvider func() (string, error)) (*Client, error) {
	return New(addrAPI, reqTimeout, func(r *gentleman.Request) (*gentleman.Request, error) {
		var accessToken, err = tokenProvider()
		if err != nil {
			return nil, err
		}
		var host string
		host, err = extractHostFromToken(accessToken)
		if err != nil {
			return nil, err
		}
		r = r.SetHeader("Authorization", "Bearer "+accessToken)
		r = r.SetHeader("X-Forwarded-Proto", "https")
		r.Context.Request.Host = host
		return r, nil
	})
}

// SetAccessToken creates a plugin to set an access token which is a valid token
func SetAccessToken(accessToken string) plugin.Plugin {
	var plugin, _ = SetAccessTokenE(accessToken)
	return plugin
}

// SetAccessTokenE creates a plugin to set an access token
func SetAccessTokenE(accessToken string) (plugin.Plugin, error) {
	host, err := extractHostFromToken(accessToken)
	if err != nil {
		return nil, err
	}

	return plugin.NewRequestPlugin(func(ctx *context.Context, h context.Handler) {
		ctx.Request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))
		ctx.Request.Header.Set("X-Forwarded-Proto", "https")
		ctx.Request.Host = host
		h.Next(ctx)
	}), nil
}

func extractHostFromToken(token string) (string, error) {
	issuer, err := extractIssuerFromToken(token)
	if err != nil {
		return "", err
	}

	var u *url.URL
	{
		var err error
		u, err = url.Parse(issuer)
		if err != nil {
			return "", errors.Wrap(err, MsgErrCannotParse+"."+PrmTokenProviderURL)
		}
	}

	return u.Host, nil
}

func extractIssuerFromToken(token string) (string, error) {
	payload, _, err := jwt.Parse(token)
	if err != nil {
		return "", errors.Wrap(err, MsgErrCannotParse+"."+PrmTokenMsg)
	}

	var jot Token
	if err = jwt.Unmarshal(payload, &jot); err != nil {
		return "", errors.Wrap(err, MsgErrCannotUnmarshal+"."+PrmTokenMsg)
	}

	return jot.Issuer, nil
}
