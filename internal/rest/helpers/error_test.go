package helpers

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestHTTPErrorHandler(t *testing.T) {
	t.Parallel()

	type testError struct {
		Message string
	}

	t.Run("invalid values", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(""))
		response := httptest.NewRecorder()
		ctx := echo.New().NewContext(request, response)
		errMessage := "Invalid values"
		he := echo.HTTPError{
			Code:     http.StatusBadRequest,
			Message:  errMessage,
			Internal: nil,
		}

		expectedBody := fmt.Sprintf(
			`{"success":false,"data":null,"error":{"code":%v,"message":"%v"}}`,
			http.StatusBadRequest,
			errMessage,
		)

		HTTPErrorHandler(&he, ctx)

		assert.Equal(t, http.StatusBadRequest, response.Code)
		assert.Equal(t, expectedBody, strings.TrimSpace(response.Body.String()))
	})

	t.Run("internal server error", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(""))
		response := httptest.NewRecorder()
		ctx := echo.New().NewContext(request, response)
		errMessage := "Internal server error"
		err := errors.New(errMessage)

		expectedBody := fmt.Sprintf(
			`{"success":false,"data":null,"error":{"code":%v,"message":"%v"}}`,
			http.StatusInternalServerError,
			err.Error(),
		)

		HTTPErrorHandler(err, ctx)

		assert.Equal(t, http.StatusInternalServerError, response.Code)
		assert.Equal(t, expectedBody, strings.TrimSpace(response.Body.String()))
	})

	t.Run("sprintf error conversion", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(""))
		response := httptest.NewRecorder()
		ctx := echo.New().NewContext(request, response)
		errMessage := "Sprintf error conversion"
		err := echo.HTTPError{
			Code:     http.StatusBadRequest,
			Message:  testError{Message: errMessage},
			Internal: nil,
		}

		expectedBody := fmt.Sprintf(
			`{"success":false,"data":null,"error":{"code":%v,"message":"{%v}"}}`,
			http.StatusBadRequest,
			errMessage,
		)

		HTTPErrorHandler(&err, ctx)

		assert.Equal(t, http.StatusBadRequest, response.Code)
		assert.Equal(t, expectedBody, strings.TrimSpace(response.Body.String()))
	})

	t.Run("internal error", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(""))
		response := httptest.NewRecorder()
		ctx := echo.New().NewContext(request, response)
		errCode := http.StatusNotFound
		errMessage := "Internal not found error"
		internalError := echo.HTTPError{
			Code:    errCode,
			Message: errMessage,
		}
		he := echo.HTTPError{
			Code:     http.StatusBadRequest,
			Message:  "Internal error",
			Internal: &internalError,
		}

		expectedBody := fmt.Sprintf(
			`{"success":false,"data":null,"error":{"code":%v,"message":"%v"}}`,
			internalError.Code,
			internalError.Message,
		)

		HTTPErrorHandler(&he, ctx)

		assert.Equal(t, internalError.Code, response.Code)
		assert.Equal(t, expectedBody, strings.TrimSpace(response.Body.String()))
	})

	t.Run("request method head", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodHead, "/", strings.NewReader(""))
		response := httptest.NewRecorder()
		ctx := echo.New().NewContext(request, response)
		errCode := http.StatusBadRequest
		errMessage := "Invalid values"
		err := echo.HTTPError{
			Code:     errCode,
			Message:  errMessage,
			Internal: nil,
		}

		HTTPErrorHandler(&err, ctx)

		assert.Equal(t, errCode, response.Code)
		assert.Empty(t, strings.TrimSpace(response.Body.String()))
	})
}
