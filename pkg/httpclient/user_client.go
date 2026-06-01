package httpclient

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/abdullahPrasetio/wapgo/internal/domain/entity"
)

// ErrUserNotFound is returned by UserClient.GetUser when the remote service
// reports that the requested user does not exist (HTTP 404).
var ErrUserNotFound = errors.New("user not found")

// UserClient implements domain/service.ExternalUserService by calling the
// remote user microservice over HTTP.
type UserClient struct {
	client  *Client
	baseURL string
}

// NewUserClient creates a UserClient pointing at baseURL (USER_SERVICE_URL config key).
// opts follows the same defaults as New — zero values are replaced with safe defaults.
func NewUserClient(baseURL string, opts Options) *UserClient {
	return &UserClient{client: New(opts), baseURL: baseURL}
}

// GetUser fetches a single user by ID from the remote user service.
// The remote service must return the standard wapgo envelope:
//
//	{"status": true, "data": { ...user fields... }}
func (c *UserClient) GetUser(ctx context.Context, id string) (*entity.User, error) {
	url := fmt.Sprintf("%s/users/%s", c.baseURL, id)

	resp, body, err := c.client.Get(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("user_client: get user %s: %w", id, err)
	}

	switch resp.StatusCode {
	case http.StatusOK:
		// handled below
	case http.StatusNotFound:
		return nil, ErrUserNotFound
	default:
		return nil, fmt.Errorf("user_client: unexpected status %d for user %s", resp.StatusCode, id)
	}

	var envelope struct {
		Data entity.User `json:"data"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return nil, fmt.Errorf("user_client: decode response: %w", err)
	}
	return &envelope.Data, nil
}
