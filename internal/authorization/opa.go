package authorization

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
)

type OPAPolicyRequest struct {
	Input any `json:"input"`
}

type OPAPolicyBooleanResponse struct {
	Result bool `json:"result"`
}

func (s *service) handleOPARequest(ctx context.Context, path string, request any) (bool, error) {
	if s.config.Disabled {
		return true, nil
	}

	var res OPAPolicyBooleanResponse
	resp, err := s.client.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		SetBody(OPAPolicyRequest{Input: request}).
		SetResult(&res).
		Post(path)
	if err != nil {
		return false, errors.Wrap(err, "request to authorization service returned error")
	}

	if resp.IsError() {
		return false, fmt.Errorf("failed to get response from authorization service: %d", resp.StatusCode())
	}

	return res.Result, nil
}
