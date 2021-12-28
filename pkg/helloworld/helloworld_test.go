package helloworld

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHandler(t *testing.T) {
	r, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	handler := http.HandlerFunc(Handler)

	handler.ServeHTTP(w, r)

	assert.Equal(t, http.StatusOK, w.Code, "Handler returned wrong status code")
	assert.Equal(t, "Hello World", w.Body.String(), "Handler returned unexpected body")
}
