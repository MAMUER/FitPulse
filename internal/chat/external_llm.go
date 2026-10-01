// Package chat provides rule-based FAQ, chat functionality, and optional external LLM integration.
package chat

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// ExternalLLMClient defines the interface for external LLM providers.
type ExternalLLMClient interface {
	// Send sends a message to the external LLM and returns the response.
	Send(ctx context.Context, userID, message string) (string, error)
}

// ExternalLLMConfig holds configuration for external LLM integration.
type ExternalLLMConfig struct {
	Enabled          bool
	Provider         string
	APIKey           string
	Model            string
	Timeout          time.Duration
	MaxTokens        int
	Temperature      float64
	AllowedUserIDs   []string // Empty means all users can use it
}

// ExternalLLMService manages external LLM interactions with fallback to rule-based chat.
type ExternalLLMService struct {
	config ExternalLLMConfig
	client ExternalLLMClient
	log    *zap.Logger
}

// NewExternalLLMService creates a new external LLM service.
// Returns nil if external LLM is not enabled.
func NewExternalLLMService(cfg ExternalLLMConfig, log *zap.Logger) *ExternalLLMService {
	if !cfg.Enabled {
		log.Info("External LLM is disabled")
		return nil
	}

	log.Info("External LLM service initialized",
		zap.String("provider", cfg.Provider),
		zap.String("model", cfg.Model),
	)

	return &ExternalLLMService{
		config: cfg,
		log:    log,
	}
}

// SetClient sets the external LLM client for the service.
func (s *ExternalLLMService) SetClient(client ExternalLLMClient) {
	s.client = client
}

// IsUserAllowed checks if a user is allowed to use the external LLM.
func (s *ExternalLLMService) IsUserAllowed(userID string) bool {
	if len(s.config.AllowedUserIDs) == 0 {
		return true
	}
	for _, id := range s.config.AllowedUserIDs {
		if id == userID {
			return true
		}
	}
	return false
}

// QueryWithFallback queries the external LLM with rule-based fallback.
// If the external LLM fails or is not allowed, it falls back to the rule-based FAQ.
func (s *ExternalLLMService) QueryWithFallback(ctx context.Context, userID, message string) (string, error) {
	if s == nil || s.client == nil {
		return FindAnswer(message), nil
	}

	if !s.IsUserAllowed(userID) {
		return FindAnswer(message), nil
	}

	response, err := s.client.Send(ctx, userID, message)
	if err != nil {
		s.log.Warn("External LLM query failed, falling back to rule-based",
			zap.Error(err),
			zap.String("user_id", userID),
		)
		return FindAnswer(message), nil
	}

	if response == "" {
		return FindAnswer(message), nil
	}

	return response, nil
}

// ExternalLLMClientImpl is a stub implementation of ExternalLLMClient.
// In a real implementation, this would integrate with OpenAI, Claude, or other LLM providers.
type ExternalLLMClientImpl struct {
	apiKey string
	model  string
}

// NewExternalLLMClientImpl creates a new external LLM client stub.
func NewExternalLLMClientImpl(apiKey, model string) *ExternalLLMClientImpl {
	return &ExternalLLMClientImpl{
		apiKey: apiKey,
		model:  model,
	}
}

// Send sends a message to the external LLM.
// This is a stub implementation that returns a placeholder response.
func (c *ExternalLLMClientImpl) Send(ctx context.Context, userID, message string) (string, error) {
	// Stub implementation - in real life this would call an external LLM API
	// For now, return a placeholder message
	return fmt.Sprintf("[External LLM (%s)] Response to: %s", c.model, message), nil
}
