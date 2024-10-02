package controllers

import (
	"github.com/UNIwise/go-template/internal/authorization"
	"github.com/UNIwise/go-template/internal/rest/contexts"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

type Handlers struct {
	authorizationService authorization.Service
}

func Register(e *echo.Group, log *logrus.Entry, authorizationService authorization.Service) *Handlers {
	h := &Handlers{
		authorizationService: authorizationService,
	}

	authCtx := contexts.AuthenticatedContextFactory(log)

	group := e.Group("/v1")

	group.POST("/dostuff", authCtx(h.dostuff))
	// group.GET("/flows", authCtx(h.getPaginatedFlowsForLicense))
	// group.GET("/flows/:id", authCtx(h.getFlow))
	// group.DELETE("/flows/:id", authCtx(h.deleteFlow))

	return h
}
