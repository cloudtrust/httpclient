package httpclient

import (
	"errors"
	"testing"
	"time"

	"github.com/cloudtrust/httpclient/mock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestNewMultiRealmTokenClient(t *testing.T) {
	var mockCtrl = gomock.NewController(t)
	defer mockCtrl.Finish()

	var mockTokenProvider = mock.NewOidcTokenProvider((mockCtrl))
	var tokenError = errors.New("token error")
	var realm = "my-realm"

	t.Run("Invalid URL", func(t *testing.T) {
		var _, err = NewMultiRealmTokenClient(":/\000/", time.Minute, mockTokenProvider)
		assert.NotNil(t, err)
	})

	var client, err = NewMultiRealmTokenClient("http://localhost", time.Minute, mockTokenProvider)
	assert.Nil(t, err)

	t.Run("GET", func(t *testing.T) {
		t.Run("can't get token", func(t *testing.T) {
			mockTokenProvider.EXPECT().ProvideToken(gomock.Any()).Return("", tokenError)
			var err = client.Get(nil)
			assert.Equal(t, tokenError, err)
		})
		t.Run("success", func(t *testing.T) {
			mockTokenProvider.EXPECT().ProvideToken(gomock.Any()).Return("default-token", nil)
			var err = client.Get(nil)
			assert.NotEqual(t, tokenError, err)
		})
	})
	t.Run("POST", func(t *testing.T) {
		t.Run("can't get token", func(t *testing.T) {
			mockTokenProvider.EXPECT().ProvideTokenForRealm(gomock.Any(), realm).Return("", tokenError)
			var _, err = client.ForRealm(realm).Post(nil)
			assert.Equal(t, tokenError, err)
		})
		t.Run("success", func(t *testing.T) {
			mockTokenProvider.EXPECT().ProvideTokenForRealm(gomock.Any(), realm).Return("token-for-"+realm, nil)
			var _, err = client.ForRealm(realm).Post(nil)
			assert.NotEqual(t, tokenError, err)
		})
	})
	t.Run("DELETE", func(t *testing.T) {
		t.Run("can't get token", func(t *testing.T) {
			mockTokenProvider.EXPECT().ProvideTokenForRealm(gomock.Any(), realm).Return("", tokenError)
			var err = client.ForRealm(realm).Delete()
			assert.Equal(t, tokenError, err)
		})
		t.Run("success", func(t *testing.T) {
			mockTokenProvider.EXPECT().ProvideTokenForRealm(gomock.Any(), realm).Return("token-for-"+realm, nil)
			var err = client.ForRealm(realm).Delete()
			assert.NotEqual(t, tokenError, err)
		})
	})
	t.Run("PUT", func(t *testing.T) {
		t.Run("can't get token", func(t *testing.T) {
			mockTokenProvider.EXPECT().ProvideTokenForRealm(gomock.Any(), realm).Return("", tokenError)
			var err = client.ForRealm(realm).Put()
			assert.Equal(t, tokenError, err)
		})
		t.Run("success", func(t *testing.T) {
			mockTokenProvider.EXPECT().ProvideTokenForRealm(gomock.Any(), realm).Return("token-for-"+realm, nil)
			var err = client.ForRealm(realm).Put()
			assert.NotEqual(t, tokenError, err)
		})
	})
}
