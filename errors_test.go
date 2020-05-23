package httpclient

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestError(t *testing.T) {
	var err = HTTPError{StatusCode: http.StatusNotFound, Message: "Where is it ?"}
	assert.False(t, err.IsSuccess())
	assert.True(t, err.IsError())
	assert.True(t, err.IsErrorFromClient())
	assert.False(t, err.IsErrorFromServer())
	assert.Equal(t, http.StatusNotFound, err.Status())
	assert.Equal(t, "Where is it ?", err.ErrorMessage())
	assert.Equal(t, "404:Where is it ?", err.Error())
}
