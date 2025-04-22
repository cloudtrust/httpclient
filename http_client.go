package httpclient

import (
	"encoding/json"
	"strings"
	"time"

	"fmt"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
	"gopkg.in/h2non/gentleman.v2"
	"gopkg.in/h2non/gentleman.v2/plugin"
	"gopkg.in/h2non/gentleman.v2/plugins/query"
	"gopkg.in/h2non/gentleman.v2/plugins/timeout"
)

// Client is the HTTP client.
type Client struct {
	apiURL      *url.URL
	httpClient  *gentleman.Client
	reqUpdaters []func(*gentleman.Request) (*gentleman.Request, error)
}

// New returns a keycloak client.
func New(addrAPI string, reqTimeout time.Duration, reqUpdaters ...func(*gentleman.Request) (*gentleman.Request, error)) (*Client, error) {
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
		apiURL:      uAPI,
		httpClient:  httpClient,
		reqUpdaters: reqUpdaters,
	}

	return client, nil
}

// applyPlugins apply all the plugins to the request req, apply also includes internal reqUpdaters
func (c *Client) applyPlugins(req *gentleman.Request, plugins ...plugin.Plugin) (*gentleman.Request, error) {
	var err error
	for _, p := range plugins {
		req = req.Use(p)
	}
	for _, updater := range c.reqUpdaters {
		req, err = updater(req)
		if err != nil {
			return nil, err
		}
	}
	return req, nil
}

func (c *Client) checkError(resp *internalResponse) error {
	switch {
	case resp.StatusCode() == http.StatusUnauthorized:
		return HTTPError{
			StatusCode: resp.StatusCode(),
			Message:    string(resp.Bytes()),
		}
	case resp.StatusCode() >= 400:
		return treatErrorStatus(resp)
	case resp.StatusCode() >= 200:
		return nil
	default:
		return HTTPError{
			StatusCode: resp.StatusCode(),
			Message:    string(resp.Bytes()),
		}
	}
}

func (c *Client) readContent(resp *internalResponse, data any) (retError error) {
	defer func() {
		if err := recover(); err != nil {
			retError = fmt.Errorf("Unexpected panic. Ensure data is declared with the expected type: %v", err)
		}
	}()
	var hdr = resp.GetHeader("Content-Type")
	switch strings.Split(hdr, ";")[0] {
	case "application/json":
		retError = resp.JSON(data)
	case "text/plain":
		*(data.(*string)) = resp.String()
		retError = nil
	case "text/html":
		*(data.(*string)) = resp.String()
		retError = nil
	case "application/octet-stream", "application/zip", "application/pdf", "text/xml":
		*(data.(*[]byte)) = resp.Bytes()
		retError = nil
	default:
		if len(resp.Bytes()) == 0 {
			retError = nil
		} else {
			retError = fmt.Errorf("%s.%v", MsgErrUnkownHTTPContentType, hdr)
		}
	}
	return retError
}

// Get is a HTTP GET method.
func (c *Client) Get(data any, plugins ...plugin.Plugin) error {
	var err error
	var req = c.httpClient.Get()
	req, err = c.applyPlugins(req, plugins...)
	if err != nil {
		return err
	}

	var gresp *gentleman.Response
	{
		gresp, err = req.Do()
		if err != nil {
			return errors.Wrap(err, MsgErrCannotObtain+"."+PrmResponse)
		}

		var resp = buildInternalResponse(gresp)
		err = c.checkError(resp)
		if err != nil {
			return err
		}
		return c.readContent(resp, data)
	}
}

// Post is a HTTP POST method
func (c *Client) Post(data any, plugins ...plugin.Plugin) (string, error) {
	var err error
	var req = c.httpClient.Post()
	req, err = c.applyPlugins(req, plugins...)
	if err != nil {
		return "", err
	}

	var gresp *gentleman.Response
	{
		gresp, err = req.Do()
		if err != nil {
			return "", errors.Wrap(err, MsgErrCannotObtain+"."+PrmResponse)
		}
		var resp = buildInternalResponse(gresp)

		err = c.checkError(resp)
		if err != nil {
			return "", err
		}
		return resp.GetHeader("Location"), c.readContent(resp, data)
	}
}

// Delete is a HTTP DELETE method
func (c *Client) Delete(plugins ...plugin.Plugin) error {
	var err error
	var req = c.httpClient.Delete()
	req, err = c.applyPlugins(req, plugins...)
	if err != nil {
		return err
	}

	var resp *gentleman.Response
	{
		resp, err = req.Do()
		if err != nil {
			return errors.Wrap(err, MsgErrCannotObtain+"."+PrmResponse)
		}

		return c.checkError(buildInternalResponse(resp))
	}
}

// Put is a HTTP PUT method
func (c *Client) Put(plugins ...plugin.Plugin) error {
	var err error
	var req = c.httpClient.Put()
	req, err = c.applyPlugins(req, plugins...)
	if err != nil {
		return err
	}

	var resp *gentleman.Response
	{
		resp, err = req.Do()
		if err != nil {
			return errors.Wrap(err, MsgErrCannotObtain+"."+PrmResponse)
		}

		return c.checkError(buildInternalResponse(resp))
	}
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

func treatErrorStatus(resp *internalResponse) error {
	var response map[string]any
	err := json.Unmarshal(resp.Bytes(), &response)
	if message, ok := response["errorMessage"]; ok && err == nil {
		return HTTPError{
			StatusCode: resp.StatusCode(),
			Message:    message.(string),
		}
	}
	return HTTPError{
		StatusCode: resp.StatusCode(),
		Message:    string(resp.Bytes()),
	}
}
