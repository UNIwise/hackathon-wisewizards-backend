package helpers

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
)

type APIResponse struct {
	Success    bool            `json:"success"`
	Data       interface{}     `json:"data"`
	Pagination *PaginationData `json:"pagination,omitempty"`
	Error      *APIError       `json:"error"`
}

type PaginationData struct {
	Offset int `json:"offset"`
	Limit  int `json:"limit"`
	Total  int `json:"total"`
}

type APIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func HTTPErrorHandler(err error, c echo.Context) {
	var he *echo.HTTPError
	if errors.As(err, &he) {
		if he.Internal != nil {
			var herr *echo.HTTPError
			if errors.As(he.Internal, &herr) {
				he = herr
			}
		}
	} else {
		he = &echo.HTTPError{
			Code:    http.StatusInternalServerError,
			Message: errors.Cause(err).Error(),
		}
	}

	code := he.Code
	var message string
	if msg, ok := he.Message.(string); ok {
		message = msg
	} else {
		message = fmt.Sprintf("%v", he.Message)
	}

	if !c.Response().Committed {
		if c.Request().Method == http.MethodHead {
			err = c.NoContent(he.Code)
		} else {
			err = c.JSON(code, &APIResponse{
				Success: false,
				Data:    nil,
				Error: &APIError{
					Code:    code,
					Message: message,
				},
			})
		}
		if err != nil {
			c.Echo().Logger.Error(err)
		}
	}
}
