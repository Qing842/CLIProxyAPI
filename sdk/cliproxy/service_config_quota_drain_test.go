package cliproxy

import (
	"testing"

	internalconfig "github.com/router-for-me/CLIProxyAPI/v8/internal/config"
	coreauth "github.com/router-for-me/CLIProxyAPI/v8/sdk/cliproxy/auth"
)

func TestQuotaDrainRoutingSelector(t *testing.T) {
	state := normalizedRoutingRuntimeState(&internalconfig.Config{Routing: internalconfig.RoutingConfig{Strategy: "quota-drain"}})
	if state.strategy != "quota-drain" {
		t.Fatalf("strategy = %q, want quota-drain", state.strategy)
	}
	selector := newRoutingSelector(state)
	if _, ok := selector.(*coreauth.QuotaDrainSelector); !ok {
		t.Fatalf("selector = %T, want *auth.QuotaDrainSelector", selector)
	}
}
