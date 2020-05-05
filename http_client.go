package httpclient

import (
	"encoding/json"
	"time"

	"fmt"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
	"gopkg.in/h2non/gentleman.v2"
	"gopkg.in/h2non/gentleman.v2/plugin"
	"gopkg.in/h2non/gentleman.v2/plugins/query"
	"gopkg.in/h2non/gentleman.v2/plugins/timeout"

	jwt "github.com/gbrlsnchs/jwt"
)

// Client is the HTTP client.
type Client struct {
	apiURL     *url.URL
	httpClient *gentleman.Client
}

// New returns a keycloak client.
func New(addrAPI string, reqTimeout time.Duration) (*Client, error) {
	var uAPI *url.URL
	{
		var err error
		uAPI, err = url.Parse(addrAPI)
		if err != nil {
			return nil, errors.Wrap(err, MsgErrCannotParse+"."+PrmAPIURL)
		}
	}

	var httpClient = gentleman.New()
	{
		httpClient = httpClient.URL(uAPI.String())
		httpClient = httpClient.Use(timeout.Request(reqTimeout))
	}

	var client = &Client{
		apiURL:     uAPI,
		httpClient: httpClient,
	}

	return client, nil
}

// Get is a HTTP GET method.
func (c *Client) Get(accessToken string, data interface{}, plugins ...plugin.Plugin) error {
	var err error
	var req = c.httpClient.Get()
	req = applyPlugins(req, plugins...)
	req, err = setAuthorisationAndHostHeaders(req, accessToken)

	if err != nil {
		return err
	}

	var resp *gentleman.Response
	{
		var err error
		resp, err = req.Do()
		if err != nil {
			return errors.Wrap(err, MsgErrCannotObtain+"."+PrmResponse)
		}

		switch {
		case resp.StatusCode == http.StatusUnauthorized:
			return HTTPError{
				StatusCode: resp.StatusCode,
				Message:    string(resp.Bytes()),
			}
		case resp.StatusCode >= 400:
			return treatErrorStatus(resp)
		case resp.StatusCode >= 200:
			switch resp.Header.Get("Content-Type") {
			case "application/json":
				return resp.JSON(data)
			case "application/octet-stream":
				data = resp.Bytes()
				return nil
			default:
				return fmt.Errorf("%s.%v", MsgErrUnkownHTTPContentType, resp.Header.Get("Content-Type"))
			}
		default:
			return fmt.Errorf("%s.%v", MsgErrUnknownResponseStatusCode, resp.StatusCode)
		}
	}
}

// Post is a HTTP POST method
func (c *Client) Post(accessToken string, data interface{}, plugins ...plugin.Plugin) (string, error) {
	var err error
	var req = c.httpClient.Post()
	req = applyPlugins(req, plugins...)
	req, err = setAuthorisationAndHostHeaders(req, accessToken)

	if err != nil {
		return "", err
	}

	var resp *gentleman.Response
	{
		var err error
		resp, err = req.Do()
		if err != nil {
			return "", errors.Wrap(err, MsgErrCannotObtain+"."+PrmResponse)
		}

		switch {
		case resp.StatusCode == http.StatusUnauthorized:
			return "", HTTPError{
				StatusCode: resp.StatusCode,
				Message:    string(resp.Bytes()),
			}
		case resp.StatusCode >= 400:
			return "", treatErrorStatus(resp)
		case resp.StatusCode >= 200:
			var location = resp.Header.Get("Location")

			switch resp.Header.Get("Content-Type") {
			case "application/json":
				return location, resp.JSON(data)
			case "application/octet-stream":
				data = resp.Bytes()
				return location, nil
			default:
				return location, nil
			}
		default:
			return "", fmt.Errorf("%s.%v", MsgErrUnknownResponseStatusCode, resp.StatusCode)
		}
	}
}

// Delete is a HTTP DELETE method
func (c *Client) Delete(accessToken string, plugins ...plugin.Plugin) error {
	var err error
	var req = c.httpClient.Delete()
	req = applyPlugins(req, plugins...)
	req, err = setAuthorisationAndHostHeaders(req, accessToken)

	if err != nil {
		return err
	}

	var resp *gentleman.Response
	{
		var err error
		resp, err = req.Do()
		if err != nil {
			return errors.Wrap(err, MsgErrCannotObtain+"."+PrmResponse)
		}

		switch {
		case resp.StatusCode == http.StatusUnauthorized:
			return HTTPError{
				StatusCode: resp.StatusCode,
				Message:    string(resp.Bytes()),
			}
		case resp.StatusCode >= 400:
			return treatErrorStatus(resp)
		case resp.StatusCode >= 200:
			return nil
		default:
			return HTTPError{
				StatusCode: resp.StatusCode,
				Message:    string(resp.Bytes()),
			}
		}
	}
}

// Put is a HTTP PUT method
func (c *Client) Put(accessToken string, plugins ...plugin.Plugin) error {
	var err error
	var req = c.httpClient.Put()
	req = applyPlugins(req, plugins...)
	req, err = setAuthorisationAndHostHeaders(req, accessToken)

	if err != nil {
		return err
	}

	var resp *gentleman.Response
	{
		var err error
		resp, err = req.Do()
		if err != nil {
			return errors.Wrap(err, MsgErrCannotObtain+"."+PrmResponse)
		}

		switch {
		case resp.StatusCode == http.StatusUnauthorized:
			return HTTPError{
				StatusCode: resp.StatusCode,
				Message:    string(resp.Bytes()),
			}
		case resp.StatusCode >= 400:
			return treatErrorStatus(resp)
		case resp.StatusCode >= 200:
			return nil
		default:
			return HTTPError{
				StatusCode: resp.StatusCode,
				Message:    string(resp.Bytes()),
			}
		}
	}
}

func setAuthorisationAndHostHeaders(req *gentleman.Request, accessToken string) (*gentleman.Request, error) {
	host, err := extractHostFromToken(accessToken)

	if err != nil {
		return req, err
	}

	var r = req.SetHeader("Authorization", fmt.Sprintf("Bearer %s", accessToken))
	r = r.SetHeader("X-Forwarded-Proto", "https")

	r.Context.Request.Host = host

	return r, nil
}

// applyPlugins apply all the plugins to the request req.
func applyPlugins(req *gentleman.Request, plugins ...plugin.Plugin) *gentleman.Request {
	var r = req
	for _, p := range plugins {
		r = r.Use(p)
	}
	return r
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

// CreateQueryPlugins create query parameters with the key values paramKV.
func CreateQueryPlugins(paramKV ...string) []plugin.Plugin {
	var plugins = []plugin.Plugin{}
	for i := 0; i+1 < len(paramKV); i += 2 {
		var k = paramKV[i]
		var v = paramKV[i+1]
		plugins = append(plugins, query.Add(k, v))
	}
	return plugins
}

func treatErrorStatus(resp *gentleman.Response) error {
	var response map[string]interface{}
	err := json.Unmarshal(resp.Bytes(), &response)
	if message, ok := response["errorMessage"]; ok && err == nil {
		return HTTPError{
			StatusCode: resp.StatusCode,
			Message:    message.(string),
		}
	}
	return HTTPError{
		StatusCode: resp.StatusCode,
		Message:    string(resp.Bytes()),
	}
}

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
