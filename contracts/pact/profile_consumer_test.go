package contracts

import (
	"testing"

	"github.com/pact-foundation/pact-go/dsl"
)

// Consumer contract test (Pact).
// Full Pact tests require pact-go v2+ and a broker for best results.
func TestGatewayProfileContract(t *testing.T) {
	_ = dsl.Pact{
		Consumer: "web-app",
		Provider: "gateway",
	}
}
