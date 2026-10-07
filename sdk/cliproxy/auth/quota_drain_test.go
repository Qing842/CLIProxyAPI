package auth

import (
	"context"
	"testing"
	"time"

	cliproxyexecutor "github.com/router-for-me/CLIProxyAPI/v8/sdk/cliproxy/executor"
)

func quotaDrainCapacity(now time.Time, windows ...CapacityWindow) CapacityState {
	return CapacityState{Supported: true, FetchedAt: now.Add(-time.Minute), StaleAt: now.Add(time.Hour), Windows: windows}
}

func quotaDrainWindow(remaining float64, resetAt time.Time, period time.Duration) CapacityWindow {
	return CapacityWindow{ID: "quota", RemainingPercent: remaining, UsedPercent: 100 - remaining, ResetAt: resetAt, PeriodSeconds: int64(period.Seconds()), Known: true, HardExhausted: remaining <= 0, Routing: true}
}

func TestQuotaDrainSelectorPrefersHighestSpendUrgency(t *testing.T) {
	now := time.Now()
	selector := &QuotaDrainSelector{}
	auths := []*Auth{
		{ID: "soon-large", Provider: "codex", Capacity: quotaDrainCapacity(now, quotaDrainWindow(80, now.Add(8*time.Hour), 7*24*time.Hour))},
		{ID: "sooner-small", Provider: "codex", Capacity: quotaDrainCapacity(now, quotaDrainWindow(5, now.Add(2*time.Hour), 7*24*time.Hour))},
	}
	got, err := selector.Pick(context.Background(), "codex", "gpt-5", cliproxyexecutor.Options{}, auths)
	if err != nil { t.Fatalf("Pick() error = %v", err) }
	if got == nil || got.ID != "soon-large" { t.Fatalf("Pick() = %#v, want soon-large", got) }
}

func TestQuotaDrainSelectorIncompleteBudgetFallsBackToRoundRobin(t *testing.T) {
	now := time.Now()
	selector := &QuotaDrainSelector{}
	auths := []*Auth{
		{ID: "a", Provider: "codex"},
		{ID: "b", Provider: "codex", Capacity: quotaDrainCapacity(now, quotaDrainWindow(80, now.Add(8*time.Hour), 7*24*time.Hour))},
	}
	first, err := selector.Pick(context.Background(), "codex", "gpt-5", cliproxyexecutor.Options{}, auths)
	if err != nil { t.Fatalf("first Pick() error = %v", err) }
	second, err := selector.Pick(context.Background(), "codex", "gpt-5", cliproxyexecutor.Options{}, auths)
	if err != nil { t.Fatalf("second Pick() error = %v", err) }
	if first == nil || second == nil || first.ID == second.ID { t.Fatalf("fallback picks = %#v then %#v, want rotation", first, second) }
}

func TestQuotaDrainSelectorFallsThroughExhaustedPriority(t *testing.T) {
	now := time.Now()
	selector := &QuotaDrainSelector{}
	auths := []*Auth{
		{ID: "high", Provider: "codex", Attributes: map[string]string{"priority": "10"}, Capacity: quotaDrainCapacity(now, quotaDrainWindow(0, now.Add(time.Hour), 7*24*time.Hour))},
		{ID: "low", Provider: "codex", Attributes: map[string]string{"priority": "0"}, Capacity: quotaDrainCapacity(now, quotaDrainWindow(50, now.Add(time.Hour), 7*24*time.Hour))},
	}
	got, err := selector.Pick(context.Background(), "codex", "gpt-5", cliproxyexecutor.Options{}, auths)
	if err != nil { t.Fatalf("Pick() error = %v", err) }
	if got == nil || got.ID != "low" { t.Fatalf("Pick() = %#v, want low", got) }
}

func TestSessionAffinityFailsOverOnProactiveQuotaExhaustion(t *testing.T) {
	now := time.Now()
	fallback := &QuotaDrainSelector{}
	selector := NewSessionAffinitySelector(fallback)
	t.Cleanup(selector.Stop)
	authA := &Auth{ID: "a", Provider: "codex", Capacity: quotaDrainCapacity(now, quotaDrainWindow(80, now.Add(time.Hour), 7*24*time.Hour))}
	authB := &Auth{ID: "b", Provider: "codex", Capacity: quotaDrainCapacity(now, quotaDrainWindow(40, now.Add(time.Hour), 7*24*time.Hour))}
	opts := cliproxyexecutor.Options{OriginalRequest: []byte(`{"metadata":{"user_id":"user_xxx_account__session_quota-drain"}}`)}
	first, err := selector.Pick(context.Background(), "codex", "gpt-5", opts, []*Auth{authA, authB})
	if err != nil { t.Fatalf("first Pick() error = %v", err) }
	if first == nil || first.ID != "a" { t.Fatalf("first Pick() = %#v, want a", first) }
	authA.Capacity = quotaDrainCapacity(now, quotaDrainWindow(0, now.Add(time.Hour), 7*24*time.Hour))
	second, err := selector.Pick(context.Background(), "codex", "gpt-5", opts, []*Auth{authA, authB})
	if err != nil { t.Fatalf("second Pick() error = %v", err) }
	if second == nil || second.ID != "b" { t.Fatalf("second Pick() = %#v, want b", second) }
}
