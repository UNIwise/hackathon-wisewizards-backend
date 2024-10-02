package contexts

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
)

func testAuthenticatedHandlerFunc(ctx AuthenticatedContext) error {
	return nil
}

func TestAuthenticatedHandlerFactory(t *testing.T) {
	t.Parallel()

	log, _ := test.NewNullLogger()
	entry := log.WithField("Test", "AuthenticatedHandlerFactory")
	e := echo.New()
	ctx := e.NewContext(
		httptest.NewRequest(http.MethodGet, "/test/url", nil),
		httptest.NewRecorder(),
	)

	t.Run("success", func(t *testing.T) {
		ctx.Request().Header.Set(licenseHeader, "123")

		err := AuthenticatedContextFactory(entry)(testAuthenticatedHandlerFunc)(ctx)

		assert.NoError(t, err)
	})

	t.Run("fail to parse license id", func(t *testing.T) {
		ctx.Request().Header.Set(licenseHeader, "")

		err := AuthenticatedContextFactory(entry)(testAuthenticatedHandlerFunc)(ctx)

		assert.ErrorIs(t, err, echo.ErrUnauthorized)
	})
}
