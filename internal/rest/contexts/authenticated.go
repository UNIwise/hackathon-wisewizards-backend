package contexts

import (
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

const licenseHeader = "X-Wiseflow-License-Id"

type AuthenticatedContextHandlerFunc func(ctx AuthenticatedContext) error

type AuthenticatedContext struct {
	echo.Context
	Log       *logrus.Entry
	LicenseID int
}

func AuthenticatedContextFactory(log *logrus.Entry) func(handler AuthenticatedContextHandlerFunc) func(ctx echo.Context) error {
	return func(handler AuthenticatedContextHandlerFunc) func(ctx echo.Context) error {
		return func(ctx echo.Context) error {
			return handler(AuthenticatedContext{
				Context:   ctx,
				Log:       log,
				LicenseID: 1,
			})
		}
	}
}
