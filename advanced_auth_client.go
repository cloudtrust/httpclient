package httpclient

import (
	"context"
	"time"

	"gopkg.in/h2non/gentleman.v2/plugin"
	"gopkg.in/h2non/gentleman.v2/plugins/headers"
)

// OidcTokenProvider provides OIDC tokens
type OidcTokenProvider interface {
	ProvideToken(ctx context.Context) (string, error)
	ProvideTokenForRealm(ctx context.Context, realm string) (string, error)
}

// RestClient interface
type RestClient interface {
	Get(data interface{}, plugins ...plugin.Plugin) error
	Post(data interface{}, plugins ...plugin.Plugin) (string, error)
	Delete(plugins ...plugin.Plugin) error
	Put(plugins ...plugin.Plugin) error
}

type MultiRealmTokenClient struct {
	client        *Client
	tokenProvider OidcTokenProvider
	realm         string
}

func NewMultiRealmTokenClient(addrAPI string, reqTimeout time.Duration, tokenProvider OidcTokenProvider) (*MultiRealmTokenClient, error) {
	var client, err = New(addrAPI, reqTimeout)
	if err != nil {
		return nil, err
	}
	return &MultiRealmTokenClient{
		client:        client,
		tokenProvider: tokenProvider,
		realm:         "",
	}, nil
}

func (mrtc *MultiRealmTokenClient) ForRealm(realm string) RestClient {
	return &MultiRealmTokenClient{
		client:        mrtc.client,
		tokenProvider: mrtc.tokenProvider,
		realm:         realm,
	}
}

func (mrtc *MultiRealmTokenClient) withRealmAuth(next func(pluginsWithAuth ...plugin.Plugin) (string, error), plugins ...plugin.Plugin) (string, error) {
	var token string
	var err error
	if mrtc.realm != "" {
		token, err = mrtc.tokenProvider.ProvideTokenForRealm(context.Background(), mrtc.realm)
	} else {
		token, err = mrtc.tokenProvider.ProvideToken(context.Background())
	}
	if err != nil {
		return "", err
	}
	plugins = append(plugins, headers.Set("Authorization", "Bearer "+token))
	return next(plugins...)
}

// Get is a HTTP GET method.
func (mrtc *MultiRealmTokenClient) Get(data interface{}, plugins ...plugin.Plugin) error {
	var _, err = mrtc.withRealmAuth(func(pluginsWithAuth ...plugin.Plugin) (string, error) {
		return "", mrtc.client.Get(data, pluginsWithAuth...)
	}, plugins...)
	return err
}

// Post is a HTTP POST method
func (mrtc *MultiRealmTokenClient) Post(data interface{}, plugins ...plugin.Plugin) (string, error) {
	return mrtc.withRealmAuth(func(pluginsWithAuth ...plugin.Plugin) (string, error) {
		return mrtc.client.Post(data, pluginsWithAuth...)
	}, plugins...)
}

// Delete is a HTTP DELETE method
func (mrtc *MultiRealmTokenClient) Delete(plugins ...plugin.Plugin) error {
	var _, err = mrtc.withRealmAuth(func(pluginsWithAuth ...plugin.Plugin) (string, error) {
		return "", mrtc.client.Delete(pluginsWithAuth...)
	}, plugins...)
	return err
}

// Put is a HTTP PUT method
func (mrtc *MultiRealmTokenClient) Put(plugins ...plugin.Plugin) error {
	var _, err = mrtc.withRealmAuth(func(pluginsWithAuth ...plugin.Plugin) (string, error) {
		return "", mrtc.client.Put(pluginsWithAuth...)
	}, plugins...)
	return err
}
