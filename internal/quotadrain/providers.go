package quotadrain

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	coreauth "github.com/router-for-me/CLIProxyAPI/v8/sdk/cliproxy/auth"
)

const maxQuotaResponseBytes = 2 << 20

var antigravityQuotaURLs = []string{
	"https://daily-cloudcode-pa.googleapis.com/v1internal:retrieveUserQuotaSummary",
	"https://daily-cloudcode-pa.sandbox.googleapis.com/v1internal:retrieveUserQuotaSummary",
	"https://cloudcode-pa.googleapis.com/v1internal:retrieveUserQuotaSummary",
}

func fetchProviderCapacity(ctx context.Context, manager *coreauth.Manager, auth *coreauth.Auth, provider string, now time.Time) ([]coreauth.CapacityWindow, error) {
	switch provider {
	case "antigravity":
		return fetchAntigravityCapacity(ctx, manager, auth, now)
	case "codex":
		return fetchCodexCapacity(ctx, manager, auth, now)
	case "xai":
		return fetchXAICapacity(ctx, manager, auth, now)
	default:
		return nil, fmt.Errorf("unsupported quota provider %q", provider)
	}
}

func requestJSON(ctx context.Context, manager *coreauth.Manager, auth *coreauth.Auth, method, target string, body []byte, headers http.Header) (map[string]any, error) {
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, errReq := http.NewRequestWithContext(ctx, method, target, reader)
	if errReq != nil {
		return nil, fmt.Errorf("build quota request: %w", errReq)
	}
	if headers != nil {
		req.Header = headers.Clone()
	}
	resp, errDo := manager.HttpRequest(ctx, auth, req)
	if errDo != nil {
		return nil, fmt.Errorf("quota request failed: %w", errDo)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxQuotaResponseBytes))
		return nil, fmt.Errorf("quota endpoint returned HTTP %d", resp.StatusCode)
	}
	decoder := json.NewDecoder(io.LimitReader(resp.Body, maxQuotaResponseBytes))
	decoder.UseNumber()
	var payload map[string]any
	if errDecode := decoder.Decode(&payload); errDecode != nil {
		return nil, fmt.Errorf("decode quota response: %w", errDecode)
	}
	if payload == nil {
		return nil, fmt.Errorf("quota response was empty")
	}
	return payload, nil
}

func fetchCodexCapacity(ctx context.Context, manager *coreauth.Manager, auth *coreauth.Auth, now time.Time) ([]coreauth.CapacityWindow, error) {
	headers := http.Header{
		"Content-Type": []string{"application/json"},
		"User-Agent":   []string{"codex-tui/0.154.0 (Mac OS 26.5.2; arm64) iTerm.app/3.6.11 (codex-tui; 0.154.0)"},
	}
	if accountID := codexAccountID(auth); accountID != "" {
		headers.Set("Chatgpt-Account-Id", accountID)
	}
	payload, errFetch := requestJSON(ctx, manager, auth, http.MethodGet, "https://chatgpt.com/backend-api/wham/usage", nil, headers)
	if errFetch != nil {
		return nil, errFetch
	}
	return parseCodexCapacity(payload, now)
}

func parseCodexCapacity(payload map[string]any, now time.Time) ([]coreauth.CapacityWindow, error) {
	windows := make([]coreauth.CapacityWindow, 0)
	appendRateInfo := func(prefix, label, scope string, info map[string]any, routing bool) {
		if len(info) == 0 {
			return
		}
		reached := boolValue(value(info, "limit_reached", "limitReached")) || explicitlyFalse(value(info, "allowed"))
		for _, spec := range []struct{ key, camel, name string }{{"primary_window", "primaryWindow", "primary"}, {"secondary_window", "secondaryWindow", "secondary"}} {
			window := objectValue(value(info, spec.key, spec.camel))
			if len(window) == 0 {
				continue
			}
			used, known := numberValue(value(window, "used_percent", "usedPercent"))
			if !known && reached {
				used, known = 100, true
			}
			if !known {
				continue
			}
			parsed := capacityWindow(prefix+"-"+spec.name, label+" "+spec.name, scope, used, windowResetAt(window, now), routing, reached || used >= 100)
			if seconds, ok := numberValue(value(window, "limit_window_seconds", "limitWindowSeconds")); ok && seconds > 0 {
				parsed.PeriodSeconds = int64(seconds)
			} else if minutes, ok := numberValue(value(window, "window_minutes", "windowMinutes")); ok && minutes > 0 {
				parsed.PeriodSeconds = int64(minutes * 60)
			}
			windows = append(windows, parsed)
		}
	}
	appendRateInfo("codex", "Codex", "", objectValue(value(payload, "rate_limit", "rateLimit")), true)
	for index, raw := range arrayValue(value(payload, "additional_rate_limits", "additionalRateLimits")) {
		item := objectValue(raw)
		label := stringValue(value(item, "limit_name", "limitName", "metered_feature", "meteredFeature"))
		if label != "" {
			appendRateInfo(fmt.Sprintf("codex-additional-%d", index), label, label, objectValue(value(item, "rate_limit", "rateLimit")), true)
		}
	}
	return requireRoutingWindows(windows)
}

func fetchXAICapacity(ctx context.Context, manager *coreauth.Manager, auth *coreauth.Auth, now time.Time) ([]coreauth.CapacityWindow, error) {
	headers := http.Header{
		"x-xai-token-auth":      []string{"xai-grok-cli"},
		"x-grok-client-version": []string{"1.0.44"},
		"Accept":                []string{"*/*"},
		"User-Agent":            []string{"xai-grok-workspace/1.0.44"},
	}
	if userID := xaiUserID(auth); userID != "" {
		headers.Set("x-userid", userID)
	}
	payload, errFetch := requestJSON(ctx, manager, auth, http.MethodGet, "https://cli-chat-proxy.grok.com/v1/billing?format=credits", nil, headers)
	if errFetch != nil {
		return nil, errFetch
	}
	return parseXAICapacity(payload, now)
}

func parseXAICapacity(payload map[string]any, now time.Time) ([]coreauth.CapacityWindow, error) {
	config := objectValue(value(payload, "config"))
	if len(config) == 0 {
		return nil, fmt.Errorf("xai quota response had no config")
	}
	period := objectValue(value(config, "currentPeriod", "current_period"))
	resetAt := parseAbsoluteTime(value(period, "end"), now)
	startAt := parseAbsoluteTime(value(period, "start"), now)
	periodSeconds := int64((7 * 24 * time.Hour).Seconds())
	if !startAt.IsZero() && !resetAt.IsZero() && resetAt.After(startAt) {
		periodSeconds = int64(resetAt.Sub(startAt).Seconds())
	}
	windows := make([]coreauth.CapacityWindow, 0, 4)
	if used, known := numberValue(value(config, "creditUsagePercent", "credit_usage_percent")); known {
		window := capacityWindow("xai-weekly", "Weekly credits", "", used, resetAt, true, used >= 100)
		window.PeriodSeconds = periodSeconds
		windows = append(windows, window)
	}
	for index, raw := range arrayValue(value(config, "productUsage", "product_usage")) {
		item := objectValue(raw)
		product := stringValue(value(item, "product", "name"))
		used, known := numberValue(value(item, "usagePercent", "usage_percent"))
		if product == "" || !known {
			continue
		}
		scope := xaiProductScope(product)
		if scope == "" {
			continue
		}
		window := capacityWindow(fmt.Sprintf("xai-product-%d", index), product, scope, used, resetAt, true, used >= 100)
		window.PeriodSeconds = periodSeconds
		windows = append(windows, window)
	}
	return requireRoutingWindows(windows)
}

func xaiProductScope(product string) string {
	normalized := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(strings.TrimSpace(product), " ", ""), "-", ""))
	switch {
	case strings.Contains(normalized, "grokbuild"):
		return "@group:xai-build"
	case strings.Contains(normalized, "grokimagine") || strings.Contains(normalized, "image"):
		return "@group:xai-imagine"
	default:
		return ""
	}
}

func fetchAntigravityCapacity(ctx context.Context, manager *coreauth.Manager, auth *coreauth.Auth, now time.Time) ([]coreauth.CapacityWindow, error) {
	projectID := authString(auth, "project_id", "projectId", "gemini_virtual_project")
	if projectID == "" {
		return nil, fmt.Errorf("quota project id is missing")
	}
	body, _ := json.Marshal(map[string]string{"project": projectID})
	var payload map[string]any
	var lastErr error
	for _, target := range antigravityQuotaURLs {
		payload, lastErr = requestJSON(ctx, manager, auth, http.MethodPost, target, body, http.Header{"Content-Type": []string{"application/json"}})
		if lastErr == nil {
			break
		}
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return parseAntigravityCapacity(payload, now)
}

func parseAntigravityCapacity(payload map[string]any, now time.Time) ([]coreauth.CapacityWindow, error) {
	groups := arrayValue(value(payload, "groups"))
	windows := make([]coreauth.CapacityWindow, 0)
	for groupIndex, rawGroup := range groups {
		group := objectValue(rawGroup)
		groupLabel := stringValue(value(group, "displayName", "display_name", "description"))
		scope := antigravityGroupScope(groupLabel)
		if scope == "" {
			continue
		}
		for bucketIndex, rawBucket := range arrayValue(value(group, "buckets")) {
			bucket := objectValue(rawBucket)
			remaining, known := numberValue(value(bucket, "remainingFraction", "remaining_fraction"))
			if !known {
				continue
			}
			if remaining <= 1 {
				remaining *= 100
			}
			label := stringValue(value(bucket, "displayName", "display_name", "window", "description"))
			if label == "" {
				label = groupLabel
			}
			window := capacityWindow(fmt.Sprintf("antigravity-%d-%d", groupIndex, bucketIndex), label, scope, 100-remaining, parseAbsoluteTime(value(bucket, "resetTime", "reset_time"), now), true, remaining <= 0)
			window.PeriodSeconds = quotaPeriodSeconds(label + " " + stringValue(value(bucket, "window")))
			windows = append(windows, window)
		}
	}
	return requireRoutingWindows(windows)
}

func antigravityGroupScope(label string) string {
	label = strings.ToLower(strings.TrimSpace(label))
	switch {
	case strings.Contains(label, "gemini"):
		return "@group:antigravity-gemini"
	case strings.Contains(label, "claude") || strings.Contains(label, "gpt"):
		return "@group:antigravity-claude-gpt"
	default:
		return ""
	}
}

func quotaPeriodSeconds(label string) int64 {
	label = strings.ToLower(strings.TrimSpace(label))
	switch {
	case strings.Contains(label, "five hour"), strings.Contains(label, "5 hour"), strings.Contains(label, "5h"):
		return int64((5 * time.Hour).Seconds())
	case strings.Contains(label, "week"):
		return int64((7 * 24 * time.Hour).Seconds())
	case strings.Contains(label, "day"):
		return int64((24 * time.Hour).Seconds())
	default:
		return 0
	}
}

func capacityWindow(id, label, scope string, used float64, resetAt time.Time, routing, exhausted bool) coreauth.CapacityWindow {
	used = math.Max(0, math.Min(100, used))
	return coreauth.CapacityWindow{ID: id, Label: label, ScopeModel: scope, UsedPercent: used, RemainingPercent: 100 - used, ResetAt: resetAt, Known: true, HardExhausted: exhausted, Routing: routing}
}

func requireRoutingWindows(windows []coreauth.CapacityWindow) ([]coreauth.CapacityWindow, error) {
	if len(windows) == 0 {
		return nil, fmt.Errorf("quota response had no usable routing windows")
	}
	return windows, nil
}

func objectValue(raw any) map[string]any { value, _ := raw.(map[string]any); return value }
func arrayValue(raw any) []any { value, _ := raw.([]any); return value }

func value(object map[string]any, keys ...string) any {
	for _, key := range keys {
		if object != nil {
			if found, ok := object[key]; ok { return found }
		}
	}
	return nil
}

func stringValue(raw any) string {
	switch typed := raw.(type) {
	case string:
		return strings.TrimSpace(typed)
	case json.Number:
		return typed.String()
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	default:
		return ""
	}
}

func numberValue(raw any) (float64, bool) {
	var number float64
	var err error
	switch typed := raw.(type) {
	case json.Number:
		number, err = typed.Float64()
	case float64:
		number = typed
	case float32:
		number = float64(typed)
	case int:
		number = float64(typed)
	case int64:
		number = float64(typed)
	case string:
		number, err = strconv.ParseFloat(strings.TrimSpace(strings.TrimSuffix(typed, "%")), 64)
	default:
		return 0, false
	}
	if err != nil || math.IsNaN(number) || math.IsInf(number, 0) { return 0, false }
	return number, true
}

func boolValue(raw any) bool {
	switch typed := raw.(type) {
	case bool:
		return typed
	case string:
		parsed, err := strconv.ParseBool(strings.TrimSpace(typed)); return err == nil && parsed
	default:
		return false
	}
}

func explicitlyFalse(raw any) bool {
	switch typed := raw.(type) {
	case bool:
		return !typed
	case string:
		parsed, err := strconv.ParseBool(strings.TrimSpace(typed)); return err == nil && !parsed
	default:
		return false
	}
}

func parseAbsoluteTime(raw any, now time.Time) time.Time {
	if text := stringValue(raw); text != "" {
		for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
			if parsed, err := time.Parse(layout, text); err == nil { return parsed.UTC() }
		}
	}
	number, ok := numberValue(raw)
	if !ok || number <= 0 { return time.Time{} }
	if number > 1_000_000_000_000 { return time.UnixMilli(int64(number)).UTC() }
	return time.Unix(int64(number), 0).UTC()
}

func windowResetAt(window map[string]any, now time.Time) time.Time {
	if absolute := parseAbsoluteTime(value(window, "reset_at", "resetAt", "resets_at", "resetsAt", "reset_time", "resetTime"), now); !absolute.IsZero() { return absolute }
	seconds, ok := numberValue(value(window, "reset_after_seconds", "resetAfterSeconds", "reset_in", "resetIn", "ttl"))
	if !ok || seconds <= 0 { return time.Time{} }
	return now.Add(time.Duration(seconds * float64(time.Second))).UTC()
}

func authString(auth *coreauth.Auth, keys ...string) string {
	if auth == nil { return "" }
	for _, key := range keys {
		if auth.Metadata != nil {
			if found := stringValue(auth.Metadata[key]); found != "" { return found }
		}
		if auth.Attributes != nil {
			if found := strings.TrimSpace(auth.Attributes[key]); found != "" { return found }
		}
	}
	return ""
}

func codexAccountID(auth *coreauth.Auth) string {
	if direct := authString(auth, "chatgpt_account_id", "chatgptAccountId", "account_id"); direct != "" { return direct }
	if auth == nil || auth.Metadata == nil { return "" }
	token := stringValue(auth.Metadata["id_token"])
	parts := strings.Split(token, ".")
	if len(parts) < 2 { return "" }
	payload, errDecode := base64.RawURLEncoding.DecodeString(parts[1])
	if errDecode != nil {
		payload, errDecode = base64.URLEncoding.DecodeString(parts[1])
		if errDecode != nil { return "" }
	}
	var claims map[string]any
	if json.Unmarshal(payload, &claims) != nil { return "" }
	if id := stringValue(value(claims, "chatgpt_account_id", "chatgptAccountId")); id != "" { return id }
	return stringValue(value(objectValue(value(claims, "https://api.openai.com/auth")), "chatgpt_account_id", "chatgptAccountId"))
}

func xaiUserID(auth *coreauth.Auth) string {
	if direct := authString(auth, "sub", "subject", "user_id", "userId"); direct != "" { return direct }
	if auth == nil || auth.Metadata == nil { return "" }
	for _, parent := range []string{"oauth", "user"} {
		nested := objectValue(auth.Metadata[parent])
		if found := stringValue(value(nested, "sub", "subject", "id")); found != "" { return found }
	}
	return ""
}
