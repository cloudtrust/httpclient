package httpclient

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"gopkg.in/h2non/gentleman.v2"

	"github.com/cloudtrust/httpclient/mock"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"gopkg.in/h2non/gentleman.v2/plugins/body"
	"gopkg.in/h2non/gentleman.v2/plugins/url"
)

func TestNewHttpClient(t *testing.T) {
	t.Run("InvalidURI", func(t *testing.T) {
		var _, err = New("://localhost", time.Minute)
		assert.NotNil(t, err)
	})
}

func TestNoResponse(t *testing.T) {
	var client, err = New("http://localhost:19766", time.Minute)
	assert.Nil(t, err)
	assert.NotNil(t, client)

	// No server is running: sending any request should return an error
	t.Run("Get", func(t *testing.T) {
		err = client.Get(nil, url.Path("/any/path/to/target"))
		assert.True(t, strings.Contains(err.Error(), "cannotObtain.response"))
	})
	t.Run("Post", func(t *testing.T) {
		_, err = client.Post(nil, url.Path("/any/path/to/target"))
		assert.True(t, strings.Contains(err.Error(), "cannotObtain.response"))
	})
	t.Run("Put", func(t *testing.T) {
		err = client.Put(url.Path("/any/path/to/target"), body.String("content"))
		assert.True(t, strings.Contains(err.Error(), "cannotObtain.response"))
	})
	t.Run("Delete", func(t *testing.T) {
		err = client.Delete(url.Path("/any/path/to/target"))
		assert.True(t, strings.Contains(err.Error(), "cannotObtain.response"))
	})
}

func TestRequestUpdaterFails(t *testing.T) {
	var expectedError = errors.New("request updater failure")
	var client, err = New("http://localhost", time.Minute, func(r *gentleman.Request) (*gentleman.Request, error) {
		return nil, expectedError
	})
	assert.Nil(t, err)
	assert.NotNil(t, client)

	// No server is running: sending any request should return an error
	t.Run("Get", func(t *testing.T) {
		err = client.Get(nil, url.Path("/any/path/to/target"))
		assert.Equal(t, expectedError, err)
	})
	t.Run("Post", func(t *testing.T) {
		_, err = client.Post(nil, url.Path("/any/path/to/target"))
		assert.Equal(t, expectedError, err)
	})
	t.Run("Put", func(t *testing.T) {
		err = client.Put(url.Path("/any/path/to/target"), body.String("content"))
		assert.Equal(t, expectedError, err)
	})
	t.Run("Delete", func(t *testing.T) {
		err = client.Delete(url.Path("/any/path/to/target"))
		assert.Equal(t, expectedError, err)
	})
}

func TestProcessResponse(t *testing.T) {
	var mockCtrl = gomock.NewController(t)
	defer mockCtrl.Finish()

	var mockHandler = mock.NewHandler(mockCtrl)
	var path = "/sample"

	r := mux.NewRouter()
	r.Handle(path, mockHandler)

	ts := httptest.NewServer(r)
	defer ts.Close()

	var client, _ = New(ts.URL, time.Minute, func(r *gentleman.Request) (*gentleman.Request, error) {
		return r, nil
	})

	var expectedError = HTTPError{
		StatusCode: http.StatusUnauthorized,
		Message:    "error message",
	}
	mockHandler.EXPECT().ServeHTTP(gomock.Any(), gomock.Any()).DoAndReturn(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("error message"))
	}).Times(4)
	t.Run("GET-Unauthorized", func(t *testing.T) {
		var resp string
		var err = client.Get(&resp, url.Path(path))
		assert.Equal(t, expectedError, err)
		assert.Equal(t, "", resp)
	})
	t.Run("POST-Unauthorized", func(t *testing.T) {
		var resp string
		var _, err = client.Post(&resp, url.Path(path))
		assert.Equal(t, expectedError, err)
	})
	t.Run("PUT-Unauthorized", func(t *testing.T) {
		var err = client.Put(url.Path(path))
		assert.Equal(t, expectedError, err)
	})
	t.Run("DELETE-Unauthorized", func(t *testing.T) {
		var err = client.Delete(url.Path(path))
		assert.Equal(t, expectedError, err)
	})

	expectedError = HTTPError{
		StatusCode: http.StatusBadRequest,
		Message:    "error message",
	}
	mockHandler.EXPECT().ServeHTTP(gomock.Any(), gomock.Any()).DoAndReturn(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"errorMessage":"error message"}`))
	}).Times(4)
	t.Run("GET-Internal server error", func(t *testing.T) {
		var resp string
		var err = client.Get(&resp, url.Path(path))
		assert.Equal(t, expectedError, err)
		assert.Equal(t, "", resp)
	})
	t.Run("POST-Internal server error", func(t *testing.T) {
		var resp string
		var _, err = client.Post(&resp, url.Path(path))
		assert.Equal(t, expectedError, err)
	})
	t.Run("PUT-Internal server error", func(t *testing.T) {
		var err = client.Put(url.Path(path))
		assert.Equal(t, expectedError, err)
	})
	t.Run("DELETE-Internal server error (json error message)", func(t *testing.T) {
		var err = client.Delete(url.Path(path))
		assert.Equal(t, expectedError, err)
	})

	mockHandler.EXPECT().ServeHTTP(gomock.Any(), gomock.Any()).DoAndReturn(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`error message`))
	})
	t.Run("DELETE-Internal server error (plain text error message)", func(t *testing.T) {
		var err = client.Delete(url.Path(path))
		assert.Equal(t, expectedError, err)
	})

	mockHandler.EXPECT().ServeHTTP(gomock.Any(), gomock.Any()).DoAndReturn(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Location", "the location")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`response`))
	}).Times(4)
	t.Run("GET-Success", func(t *testing.T) {
		var resp string
		var err = client.Get(&resp, url.Path(path))
		assert.Nil(t, err)
		assert.Equal(t, "response", resp)
	})
	t.Run("POST-Success", func(t *testing.T) {
		var resp string
		var location, err = client.Post(&resp, url.Path(path))
		assert.Nil(t, err)
		assert.Equal(t, "the location", location)
	})
	t.Run("PUT-Success", func(t *testing.T) {
		var err = client.Put(url.Path(path))
		assert.Nil(t, err)
	})
	t.Run("DELETE-Success", func(t *testing.T) {
		var err = client.Delete(url.Path(path))
		assert.Nil(t, err)
	})

	var response = `{"key1": "value1", "key2": 234}`
	mockHandler.EXPECT().ServeHTTP(gomock.Any(), gomock.Any()).DoAndReturn(func(w http.ResponseWriter, r *http.Request) {
		var content = r.URL.Query().Get("content")
		w.Header().Set("Content-Type", content)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(response))
	}).Times(4)
	t.Run("Content-Type text/plain", func(t *testing.T) {
		var resp string
		var plugins = CreateQueryPlugins("content", "text/plain")
		plugins = append(plugins, url.Path(path))
		var err = client.Get(&resp, plugins...)
		assert.Nil(t, err)
		assert.Equal(t, response, resp)
	})
	t.Run("Content-Type application/octet-stream", func(t *testing.T) {
		var resp []byte
		var plugins = CreateQueryPlugins("content", "application/octet-stream")
		plugins = append(plugins, url.Path(path))
		var err = client.Get(&resp, plugins...)
		assert.Nil(t, err)
		assert.Equal(t, []byte(response), resp)
	})
	t.Run("Content-Type application/json", func(t *testing.T) {
		var resp map[string]interface{}
		var plugins = CreateQueryPlugins("content", "application/json")
		plugins = append(plugins, url.Path(path))
		var err = client.Get(&resp, plugins...)
		assert.Nil(t, err)
		assert.Len(t, resp, 2)
		assert.Equal(t, "value1", resp["key1"])
	})
	t.Run("Content-Type not supported", func(t *testing.T) {
		var plugins = CreateQueryPlugins("content", "application/unknown")
		plugins = append(plugins, url.Path(path))
		var err = client.Get(nil, plugins...)
		assert.NotNil(t, err)
	})

	mockHandler.EXPECT().ServeHTTP(gomock.Any(), gomock.Any()).DoAndReturn(func(w http.ResponseWriter, r *http.Request) {
		var content = r.URL.Query().Get("content")
		w.Header().Set("Content-Type", content)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(""))
	})
	t.Run("Empty content", func(t *testing.T) {
		var plugins = CreateQueryPlugins("content", "application/unknown")
		plugins = append(plugins, url.Path(path))
		var err = client.Get(nil, plugins...)
		assert.Nil(t, err)
	})
}
