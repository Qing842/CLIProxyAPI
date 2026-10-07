package quotadrain

import (
	"testing"
	"time"
)

func TestParseAntigravityCapacitySeparatesModelGroups(t *testing.T) {
	now := time.Now()
	payload := map[string]any{"groups": []any{
		map[string]any{"displayName": "GEMINI Models", "buckets": []any{
			map[string]any{"displayName": "Five Hour Limit Remaining", "remainingFraction": 1.0, "resetTime": now.Add(4 * time.Hour).Format(time.RFC3339)},
			map[string]any{"displayName": "Weekly Limit Remaining", "remainingFraction": 0.59, "resetTime": now.Add(44 * time.Hour).Format(time.RFC3339)},
		}},
		map[string]any{"displayName": "CLAUDE and GPT Models", "buckets": []any{
			map[string]any{"displayName": "Weekly Limit Remaining", "remainingFraction": 0.96, "resetTime": now.Add(6 * 24 * time.Hour).Format(time.RFC3339)},
		}},
	}}
	windows, err := parseAntigravityCapacity(payload, now)
	if err != nil { t.Fatalf("parseAntigravityCapacity() error = %v", err) }
	if len(windows) != 3 { t.Fatalf("windows = %d, want 3", len(windows)) }
	if windows[1].ScopeModel != "@group:antigravity-gemini" || windows[1].PeriodSeconds != int64((7*24*time.Hour).Seconds()) { t.Fatalf("gemini weekly window = %#v", windows[1]) }
	if windows[2].ScopeModel != "@group:antigravity-claude-gpt" { t.Fatalf("claude/gpt window = %#v", windows[2]) }
}

func TestParseCodexCapacityScopesAdditionalLimits(t *testing.T) {
	now := time.Now()
	payload := map[string]any{
		"rate_limit": map[string]any{"primary_window": map[string]any{"used_percent": 70.0, "limit_window_seconds": float64((7 * 24 * time.Hour).Seconds()), "reset_at": float64(now.Add(2 * 24 * time.Hour).Unix())}},
		"additional_rate_limits": []any{map[string]any{"limit_name": "gpt-reserve", "rate_limit": map[string]any{"primary_window": map[string]any{"used_percent": 0.0, "limit_window_seconds": float64((7 * 24 * time.Hour).Seconds()), "reset_at": float64(now.Add(6 * 24 * time.Hour).Unix())}}}},
	}
	windows, err := parseCodexCapacity(payload, now)
	if err != nil { t.Fatalf("parseCodexCapacity() error = %v", err) }
	if len(windows) != 2 { t.Fatalf("windows = %d, want 2", len(windows)) }
	if windows[1].ScopeModel != "gpt-reserve" || !windows[1].Routing { t.Fatalf("additional window = %#v", windows[1]) }
}

func TestParseXAICapacityUsesBuildPoolForTextRouting(t *testing.T) {
	now := time.Now()
	payload := map[string]any{"config": map[string]any{
		"creditUsagePercent": 70.0,
		"currentPeriod": map[string]any{"start": now.Add(-5 * 24 * time.Hour).Format(time.RFC3339), "end": now.Add(2 * 24 * time.Hour).Format(time.RFC3339)},
		"productUsage": []any{
			map[string]any{"product": "GrokBuild", "usagePercent": 68.0},
			map[string]any{"product": "GrokChat", "usagePercent": 1.0},
			map[string]any{"product": "GrokImagine", "usagePercent": 1.0},
		},
	}}
	windows, err := parseXAICapacity(payload, now)
	if err != nil { t.Fatalf("parseXAICapacity() error = %v", err) }
	if len(windows) != 3 { t.Fatalf("windows = %d, want 3", len(windows)) }
	if windows[1].ScopeModel != "@group:xai-build" || windows[2].ScopeModel != "@group:xai-imagine" { t.Fatalf("product scopes = %#v / %#v", windows[1], windows[2]) }
}
