//go:generate mockgen --source=service.go -destination=service_mock.go -package=authorization -mock_names Service=MockService
package authorization

import (
	"context"

	"github.com/go-resty/resty/v2"
)

type Config struct {
	DSN      string `default:"http://localhost:8181/v1/data"`
	Disabled bool   `default:"false"`
}

type Service interface {
	CanReadFlow(
		ctx context.Context,
		licenseID int,
	) (allowed bool, err error)
	CanWriteToFlow(
		ctx context.Context,
		licenseID int,
	) (allowed bool, err error)
}

type service struct {
	client *resty.Client
	config Config
}

func NewService(config Config) Service {
	client := resty.New()
	client.SetBaseURL(config.DSN)

	return &service{
		client: client,
		config: config,
	}
}

type CanWriteToFlowRequest struct {
	LicenseID int `json:"licenseId"`
}

func (s *service) CanWriteToFlow(ctx context.Context, licenseID int) (bool, error) {
	return s.handleOPARequest(ctx, "/flows/flow/write", CanWriteToFlowRequest{
		LicenseID: licenseID,
	})
}

type CanReadFlowRequest struct {
	LicenseID int `json:"licenseId"`
}

func (s *service) CanReadFlow(ctx context.Context, licenseID int) (bool, error) {
	return s.handleOPARequest(ctx, "/flows/flow/read", CanReadFlowRequest{
		LicenseID: licenseID,
	})
}
