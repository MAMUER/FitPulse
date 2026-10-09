package withings

import (
	"errors"
)

const (
	authorizeURL = "https://account.withings.com/oauth2_user/authorize2"
	tokenURL     = "https://wbsapi.withings.net/v2/oauth2"
)

type Config struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	APIBaseURL   string
	Scopes       []string
}

func DefaultScopes() []string {
	return []string{
		"user.metrics",
		"user.activity",
		"user.sleep",
		"user.menstrual_cycles",
	}
}

type TokenResponse struct {
	Status  int    `json:"status"`
	Body    struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
		TokenType    string `json:"token_type"`
		Scope        string `json:"scope"`
	} `json:"body"`
}

func (c *Config) Validate() error {
	if c.ClientID == "" {
		return errors.New("withings client_id is required")
	}
	if c.ClientSecret == "" {
		return errors.New("withings client_secret is required")
	}
	if c.RedirectURL == "" {
		return errors.New("withings redirect_url is required")
	}
	if c.APIBaseURL == "" {
		return errors.New("withings api_base_url is required")
	}
	return nil
}
