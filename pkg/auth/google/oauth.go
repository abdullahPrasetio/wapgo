package google

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/oauth2"
	googleoauth "golang.org/x/oauth2/google"
)

// Config holds Google OAuth2 credentials.
type Config struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

// UserInfo is the Google profile subset returned after a successful exchange.
type UserInfo struct {
	ID      string `json:"id"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture"`
}

// Provider wraps golang.org/x/oauth2 for the Google OAuth2 flow.
type Provider struct {
	cfg *oauth2.Config
}

// New creates a Google OAuth2 Provider.
func New(cfg Config) *Provider {
	return &Provider{
		cfg: &oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Scopes:       []string{"openid", "email", "profile"},
			Endpoint:     googleoauth.Endpoint,
		},
	}
}

// AuthURL returns the Google consent page URL with the given CSRF state token.
func (p *Provider) AuthURL(state string) string {
	return p.cfg.AuthCodeURL(state, oauth2.AccessTypeOnline)
}

// Exchange exchanges an authorization code for Google user info.
func (p *Provider) Exchange(ctx context.Context, code string) (*UserInfo, error) {
	token, err := p.cfg.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("google oauth exchange: %w", err)
	}

	client := p.cfg.Client(ctx, token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return nil, fmt.Errorf("google userinfo fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google userinfo: unexpected status %d", resp.StatusCode)
	}

	var info UserInfo
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4*1024)).Decode(&info); err != nil {
		return nil, fmt.Errorf("google userinfo decode: %w", err)
	}
	return &info, nil
}
