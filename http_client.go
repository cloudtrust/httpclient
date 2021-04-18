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

func (c *Client) checkError(resp *gentleman.Response) error {
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

func (c *Client) readContent(resp *gentleman.Response, data interface{}) error {
	var hdr = resp.Header.Get("Content-Type")
	switch strings.Split(hdr, ";")[0] {
	case "application/json":
		return resp.JSON(data)
	case "text/plain":
		*(data.(*string)) = resp.String()
		return nil
	case "text/html":
		*(data.(*string)) = resp.String()
		return nil
	case "application/octet-stream", "application/zip", "application/pdf", "text/xml":
		*(data.(*[]byte)) = resp.Bytes()
		return nil
	default:
		if len(resp.Bytes()) == 0 {
			return nil
		}
		return fmt.Errorf("%s.%v", MsgErrUnkownHTTPContentType, hdr)
	}
}

// Get is a HTTP GET method.
func (c *Client) Get(data interface{}, plugins ...plugin.Plugin) error {
	var err error
	var req = c.httpClient.Get()
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

		err = c.checkError(resp)
		if err != nil {
			return err
		}
		return c.readContent(resp, data)
	}
}

// Post is a HTTP POST method
func (c *Client) Post(data interface{}, plugins ...plugin.Plugin) (string, error) {
	var err error
	var req = c.httpClient.Post()
	req, err = c.applyPlugins(req, plugins...)
	if err != nil {
		return "", err
	}

	var resp *gentleman.Response
	{
		resp, err = req.Do()
		if err != nil {
			return "", errors.Wrap(err, MsgErrCannotObtain+"."+PrmResponse)
		}

		err = c.checkError(resp)
		if err != nil {
			return "", err
		}
		return resp.Header.Get("Location"), c.readContent(resp, data)
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

		return c.checkError(resp)
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

		return c.checkError(resp)
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
