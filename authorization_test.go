package httpclient

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cloudtrust/httpclient/mock"
	"go.uber.org/mock/gomock"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"gopkg.in/h2non/gentleman.v2/plugins/url"
)

const (
	accessTokenValid         = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyLCJpc3MiOiJodHRwczovL3NhbXBsZS5jb20vIn0.xLlV0CYqKDIPI-_IEABEcjRnKVNklivaw9WRmR8SXto"
	accessTokenInvalidIssuer = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyLCJpc3MiOiI6In0.AqSnXvu_rUxZOfgy8y42T6Pzmua-ZvqkRbC3eGXnm1A"
)

func TestInvalidBearerAuthClient(t *testing.T) {
	var expectedError = errors.New("fail")
	t.Run("Can't provide token", func(t *testing.T) {
		var client, err = NewBearerAuthClient("http://localhost", time.Minute, func() (string, error) {
			return "", expectedError
		})
		assert.Nil(t, err)
		assert.Equal(t, expectedError, client.Delete())
	})
	t.Run("Provide invalid token", func(t *testing.T) {
		var client, err = NewBearerAuthClient("http://localhost", time.Minute, func() (string, error) {
			return accessTokenInvalidIssuer, nil
		})
		assert.Nil(t, err)
		assert.NotNil(t, client.Delete())
	})
}

func TestHttpClientAuthentication(t *testing.T) {
	var mockCtrl = gomock.NewController(t)
	defer mockCtrl.Finish()

	var mockHandler = mock.NewHandler(mockCtrl)

	r := mux.NewRouter()
	r.Handle("/sample", mockHandler)

	ts := httptest.NewServer(r)
	defer ts.Close()

	var httpStatus = http.StatusNoContent

	t.Run("Simple HTTP client", func(t *testing.T) {
		httpClient, _ := New(ts.URL, time.Minute)
		mockHandler.EXPECT().ServeHTTP(gomock.Any(), gomock.Any()).DoAndReturn(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "", r.Header.Get("Authorization"))
			w.WriteHeader(httpStatus)
		}).Times(4)
		t.Run("get", func(t *testing.T) {
			var res string
			var err = httpClient.Get(&res, url.Path("/sample"))
			assert.Nil(t, err)
		})
		t.Run("post", func(t *testing.T) {
			var res string
			var _, err = httpClient.Post(&res, url.Path("/sample"))
			assert.Nil(t, err)
		})
		t.Run("put", func(t *testing.T) {
			var err = httpClient.Put(url.Path("/sample"))
			assert.Nil(t, err)
		})
		t.Run("delete", func(t *testing.T) {
			var err = httpClient.Delete(url.Path("/sample"))
			assert.Nil(t, err)
		})
	})
	t.Run("HTTP client with basic authentication", func(t *testing.T) {
		httpClient, _ := NewBasicAuthClient(ts.URL, time.Minute, "user", "pass")
		mockHandler.EXPECT().ServeHTTP(gomock.Any(), gomock.Any()).DoAndReturn(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "Basic dXNlcjpwYXNz", r.Header.Get("Authorization"))
			w.WriteHeader(httpStatus)
		}).Times(4)
		t.Run("get", func(t *testing.T) {
			var res string
			var err = httpClient.Get(&res, url.Path("/sample"))
			assert.Nil(t, err)
		})
		t.Run("post", func(t *testing.T) {
			var res string
			var _, err = httpClient.Post(&res, url.Path("/sample"))
			assert.Nil(t, err)
		})
		t.Run("put", func(t *testing.T) {
			var err = httpClient.Put(url.Path("/sample"))
			assert.Nil(t, err)
		})
		t.Run("delete", func(t *testing.T) {
			var err = httpClient.Delete(url.Path("/sample"))
			assert.Nil(t, err)
		})
	})
	t.Run("HTTP client with bearer authentication", func(t *testing.T) {
		httpClient, _ := NewBearerAuthClient(ts.URL, time.Minute, func() (string, error) {
			return accessTokenValid, nil
		})
		mockHandler.EXPECT().ServeHTTP(gomock.Any(), gomock.Any()).DoAndReturn(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "Bearer "+accessTokenValid, r.Header.Get("Authorization"))
			w.WriteHeader(httpStatus)
		}).Times(4)
		t.Run("get", func(t *testing.T) {
			var res string
			var err = httpClient.Get(&res, url.Path("/sample"))
			assert.Nil(t, err)
		})
		t.Run("post", func(t *testing.T) {
			var res string
			var _, err = httpClient.Post(&res, url.Path("/sample"))
			assert.Nil(t, err)
		})
		t.Run("put", func(t *testing.T) {
			var err = httpClient.Put(url.Path("/sample"))
			assert.Nil(t, err)
		})
		t.Run("delete", func(t *testing.T) {
			var err = httpClient.Delete(url.Path("/sample"))
			assert.Nil(t, err)
		})
	})
	t.Run("HTTP client with contextual bearer authentication", func(t *testing.T) {
		httpClient, _ := NewBearerAuthClientContext(ts.URL, time.Minute, func(ctx any) (string, error) {
			if ctx == nil {
				return "invalid", nil
			}
			var strContext = ctx.(string)
			return accessTokenValid + strContext, nil
		})
		t.Run("missing context", func(t *testing.T) {
			mockHandler.EXPECT().ServeHTTP(gomock.Any(), gomock.Any()).DoAndReturn(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "Bearer invalid", r.Header.Get("Authorization"))
				w.WriteHeader(http.StatusNoContent)
			})
			var res string
			var err = httpClient.Get(&res, url.Path("/sample"))
			assert.Nil(t, err)
		})
		t.Run("get", func(t *testing.T) {
			mockHandler.EXPECT().ServeHTTP(gomock.Any(), gomock.Any()).DoAndReturn(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "Bearer "+accessTokenValid+"mygetctx", r.Header.Get("Authorization"))
				w.WriteHeader(http.StatusNoContent)
			})
			var res string
			var err = httpClient.WithContext("mygetctx").Get(&res, url.Path("/sample"))
			assert.Nil(t, err)
		})
		t.Run("post", func(t *testing.T) {
			mockHandler.EXPECT().ServeHTTP(gomock.Any(), gomock.Any()).DoAndReturn(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "Bearer "+accessTokenValid+"mypostctx", r.Header.Get("Authorization"))
				w.WriteHeader(http.StatusNoContent)
			})
			var res string
			var _, err = httpClient.WithContext("mypostctx").Post(&res, url.Path("/sample"))
			assert.Nil(t, err)
		})
	})
	t.Run("HTTP client with access token", func(t *testing.T) {
		httpClient, _ := New(ts.URL, time.Minute)
		mockHandler.EXPECT().ServeHTTP(gomock.Any(), gomock.Any()).DoAndReturn(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "Bearer "+accessTokenValid, r.Header.Get("Authorization"))
			w.WriteHeader(httpStatus)
		}).Times(4)
		t.Run("get", func(t *testing.T) {
			var res string
			var err = httpClient.Get(&res, url.Path("/sample"), SetAccessToken(accessTokenValid))
			assert.Nil(t, err)
		})
		t.Run("post", func(t *testing.T) {
			var res string
			var _, err = httpClient.Post(&res, url.Path("/sample"), SetAccessToken(accessTokenValid))
			assert.Nil(t, err)
		})
		t.Run("put", func(t *testing.T) {
			var err = httpClient.Put(url.Path("/sample"), SetAccessToken(accessTokenValid))
			assert.Nil(t, err)
		})
		t.Run("delete", func(t *testing.T) {
			var err = httpClient.Delete(url.Path("/sample"), SetAccessToken(accessTokenValid))
			assert.Nil(t, err)
		})
	})
	t.Run("AccessToken issue-Invalid token", func(t *testing.T) {
		var accessToken = "eyJhbGcito"
		var _, err = SetAccessTokenE(accessToken)
		assert.NotNil(t, err)
	})
	t.Run("AccessToken issue-Invalid issuer", func(t *testing.T) {
		var accessToken = accessTokenInvalidIssuer
		var _, err = SetAccessTokenE(accessToken)
		assert.NotNil(t, err)
	})
}

func TestExtractIssuerFromToken(t *testing.T) {
	t.Run("Can't parse JWT", func(t *testing.T) {
		var _, err = extractIssuerFromToken("AAABBBCCC")
		assert.NotNil(t, err)
	})
	t.Run("Can't unmarshal token", func(t *testing.T) {
		var _, err = extractIssuerFromToken("AAA.BBB.CCC")
		assert.NotNil(t, err)
	})
	t.Run("Valid token", func(t *testing.T) {
		var issuer, err = extractIssuerFromToken(accessTokenValid)
		assert.Nil(t, err)
		assert.Equal(t, "https://sample.com/", issuer)
	})
}
