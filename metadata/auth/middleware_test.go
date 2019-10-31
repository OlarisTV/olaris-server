package auth

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"gitlab.com/olaris/olaris-server/metadata/app"
	"gitlab.com/olaris/olaris-server/metadata/db"
	"net/http"
	"net/http/httptest"
	"testing"
)

type TestHandler struct {
	called bool
}

func (th *TestHandler) HandlerFunc() http.HandlerFunc {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		th.called = true
	})
}

func (th *TestHandler) Called() bool {
	return th.called
}

func TestMiddleWare_InvalidToken(t *testing.T) {
	// TOOD(Leon Handreke): We need this to fill the database singleton
	app.NewTestingMDContext(nil)
	db.CreateUser("test", "testtest", false)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Add("Authorization", "blabla")

	fakeHandler := TestHandler{}
	handler := MiddleWare(fakeHandler.HandlerFunc())

	rw := httptest.NewRecorder()
	handler.ServeHTTP(rw, req)

	assert.EqualValues(t, http.StatusUnauthorized, rw.Result().StatusCode)
	assert.False(t, fakeHandler.Called())
}

func TestMiddleWare(t *testing.T) {
	// TOOD(Leon Handreke): We need this to fill the database singleton
	app.NewTestingMDContext(nil)
	user, _ := db.CreateUser("test", "testtest", false)

	tokenStr, _ := CreateMetadataJWT(&user, DefaultLoginTokenValidity)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", tokenStr))

	fakeHandler := TestHandler{}
	handler := MiddleWare(fakeHandler.HandlerFunc())

	rw := httptest.NewRecorder()
	handler.ServeHTTP(rw, req)

	assert.EqualValues(t, http.StatusOK, rw.Result().StatusCode)
	assert.True(t, fakeHandler.Called())
}
