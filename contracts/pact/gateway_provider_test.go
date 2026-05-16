package contracts

import (
	"testing"

	"github.com/pact-foundation/pact-go/dsl"
)

// Provider verification is run separately (requires pact file + running provider).
// This is a placeholder so go vet passes.
func TestGatewayProviderVerification(t *testing.T) {
	_ = dsl.Pact{Provider: "gateway"}
}
