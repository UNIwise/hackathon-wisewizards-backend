package authorization

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/jarcoal/httpmock"
	"github.com/tj/assert"
)

func TestServiceCanReadFlow(t *testing.T) {
	t.Parallel()

	licenseID := 12
	path := "/flows/flow/read"
	mockService := newMockedService(t)

	t.Run("allowed", func(t *testing.T) {
		expectRequest(
			t,
			path,
			OPAPolicyRequest{
				Input: CanReadFlowRequest{
					LicenseID: licenseID,
				},
			},
			http.StatusOK,
			OPAPolicyBooleanResponse{
				Result: true,
			},
		)

		allow, err := mockService.CanReadFlow(context.Background(), licenseID)

		assert.True(t, allow)
		assert.NoError(t, err)
	})

	t.Run("not allowed", func(t *testing.T) {
		expectRequest(
			t,
			path,
			OPAPolicyRequest{
				Input: CanReadFlowRequest{
					LicenseID: licenseID,
				},
			},
			http.StatusOK,
			OPAPolicyBooleanResponse{
				Result: false,
			},
		)

		allow, err := mockService.CanReadFlow(context.Background(), licenseID)

		assert.False(t, allow)
		assert.NoError(t, err)
	})

	t.Run("error", func(t *testing.T) {
		httpmock.RegisterResponder(
			http.MethodPost,
			path,
			httpmock.NewErrorResponder(errors.New("error")),
		)

		allow, err := mockService.CanReadFlow(context.Background(), licenseID)

		assert.False(t, allow)
		assert.Error(t, err)
	})
}

func TestServiceCanWriteToFlow(t *testing.T) {
	t.Parallel()

	licenseID := 12
	path := "/flows/flow/write"
	mockService := newMockedService(t)

	t.Run("allowed", func(t *testing.T) {
		expectRequest(
			t,
			path,
			OPAPolicyRequest{
				Input: CanWriteToFlowRequest{
					LicenseID: licenseID,
				},
			},
			http.StatusOK,
			OPAPolicyBooleanResponse{
				Result: true,
			},
		)

		allow, err := mockService.CanWriteToFlow(context.Background(), licenseID)

		assert.True(t, allow)
		assert.NoError(t, err)
	})

	t.Run("not allowed", func(t *testing.T) {
		expectRequest(
			t,
			path,
			OPAPolicyRequest{
				Input: CanReadFlowRequest{
					LicenseID: licenseID,
				},
			},
			http.StatusOK,
			OPAPolicyBooleanResponse{
				Result: false,
			},
		)

		allow, err := mockService.CanWriteToFlow(context.Background(), licenseID)

		assert.False(t, allow)
		assert.NoError(t, err)
	})

	t.Run("error", func(t *testing.T) {
		httpmock.RegisterResponder(
			http.MethodPost,
			path,
			httpmock.NewErrorResponder(errors.New("error")),
		)

		allow, err := mockService.CanWriteToFlow(context.Background(), licenseID)

		assert.False(t, allow)
		assert.Error(t, err)
	})
}

func newMockedService(t *testing.T) Service {
	client := resty.New()

	httpmock.ActivateNonDefault(client.GetClient())
	t.Cleanup(func() {
		httpmock.DeactivateAndReset()
	})

	return &service{
		client: client,
		config: Config{
			DSN:      "http://localhost:8181/v1/data",
			Disabled: false,
		},
	}
}

func expectRequest(t *testing.T, path string, body OPAPolicyRequest, statusCode int, response interface{}) {
	httpmock.RegisterResponder(
		http.MethodPost,
		path,
		func(req *http.Request) (*http.Response, error) {
			bodyBytes, err := io.ReadAll(req.Body)
			assert.NoError(t, err)

			expectedJSON, err := json.Marshal(body)
			assert.NoError(t, err)

			assert.JSONEq(t, string(expectedJSON), string(bodyBytes))

			resp, err := httpmock.NewJsonResponse(statusCode, response)
			assert.NoError(t, err)

			return resp, nil
		},
	)
}
