package withings

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultScopes(t *testing.T) {
	scopes := DefaultScopes()

	require.Len(t, scopes, 4)
	assert.Equal(t, []string{"user.metrics", "user.activity", "user.sleep", "user.menstrual_cycles"}, scopes)
}

func TestDefaultScopesReturnsIndependentCopies(t *testing.T) {
	first := DefaultScopes()
	first[0] = "mutated"

	second := DefaultScopes()
	assert.Equal(t, "user.metrics", second[0])
	assert.Equal(t, "user.activity", second[1])
}

func TestTokenURLConstants(t *testing.T) {
	assert.Equal(t, "https://account.withings.com/oauth2_user/authorize2", authorizeURL)
	assert.Equal(t, "https://wbsapi.withings.net/v2/oauth2", tokenURL)
}

func TestTokenResponseDecoding(t *testing.T) {
	raw := `{
		"status": 0,
		"body": {
			"access_token": "access-1",
			"refresh_token": "refresh-1",
			"expires_in": 10800,
			"token_type": "Bearer",
			"scope": "user.metrics user.info"
		}
	}`

	var token TokenResponse
	require.NoError(t, json.Unmarshal([]byte(raw), &token))

	assert.Equal(t, 0, token.Status)
	assert.Equal(t, "access-1", token.Body.AccessToken)
	assert.Equal(t, "refresh-1", token.Body.RefreshToken)
	assert.Equal(t, 10800, token.Body.ExpiresIn)
	assert.Equal(t, "Bearer", token.Body.TokenType)
	assert.Equal(t, "user.metrics user.info", token.Body.Scope)
}

func TestTokenResponseDecodingDefaults(t *testing.T) {
	var token TokenResponse
	require.NoError(t, json.Unmarshal([]byte(`{"status": 7}`), &token))

	assert.Equal(t, 7, token.Status)
	assert.Empty(t, token.Body.AccessToken)
	assert.Empty(t, token.Body.RefreshToken)
	assert.Equal(t, 0, token.Body.ExpiresIn)
}

func TestConfigValidate(t *testing.T) {
	valid := Config{
		ClientID:     "client-id",
		ClientSecret: "client-secret",
		RedirectURL:  "https://fitpulse.example/callback",
		APIBaseURL:   "https://wbsapi.withings.net",
		Scopes:       DefaultScopes(),
	}
	require.NoError(t, valid.Validate())

	cases := []struct {
		name    string
		mutate  func(*Config)
		message string
	}{
		{"missing client id", func(c *Config) { c.ClientID = "" }, "withings client_id is required"},
		{"missing client secret", func(c *Config) { c.ClientSecret = "" }, "withings client_secret is required"},
		{"missing redirect url", func(c *Config) { c.RedirectURL = "" }, "withings redirect_url is required"},
		{"missing api base url", func(c *Config) { c.APIBaseURL = "" }, "withings api_base_url is required"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := Config{
				ClientID:     "client-id",
				ClientSecret: "client-secret",
				RedirectURL:  "https://fitpulse.example/callback",
				APIBaseURL:   "https://wbsapi.withings.net",
			}
			tc.mutate(&cfg)

			err := cfg.Validate()
			require.Error(t, err)
			assert.EqualError(t, err, tc.message)
		})
	}

	t.Run("zero value fails validation", func(t *testing.T) {
		var cfg Config
		assert.Error(t, cfg.Validate())
	})

	t.Run("scopes are not required", func(t *testing.T) {
		cfg := Config{
			ClientID:     "client-id",
			ClientSecret: "client-secret",
			RedirectURL:  "https://fitpulse.example/callback",
			APIBaseURL:   "https://wbsapi.withings.net",
		}
		assert.NoError(t, cfg.Validate())
	})
}

func TestV2EndpointConstants(t *testing.T) {
	assert.Equal(t, "/v2/measure", v2Measure)
	assert.Equal(t, "/v2/sleep", v2Sleep)
	assert.Equal(t, "/v2/activity", v2Activity)
}
