package auth

import (
	"math"
	"sort"
	"strings"
	"time"
)

const (
	quotaCapacityPercentEpsilon = 0.0001
	quotaBudgetWindowMinPeriod  = 24 * time.Hour
	quotaMinimumUrgencyHorizon  = time.Hour
	quotaCapacityUrgencyEpsilon = 1e-9
)

const (
	quotaScopeAntigravityGemini    = "@group:antigravity-gemini"
	quotaScopeAntigravityClaudeGPT = "@group:antigravity-claude-gpt"
	quotaScopeXAIBuild              = "@group:xai-build"
	quotaScopeXAIImagine            = "@group:xai-imagine"
)

type CapacityState struct {
	Provider      string           `json:"provider"`
	Supported     bool             `json:"supported"`
	FetchedAt     time.Time        `json:"fetched_at,omitempty,omitzero"`
	StaleAt       time.Time        `json:"stale_at,omitempty,omitzero"`
	LastAttemptAt time.Time        `json:"last_attempt_at,omitempty,omitzero"`
	LastError     string           `json:"last_error,omitempty"`
	Windows       []CapacityWindow `json:"windows,omitempty"`
}

type CapacityWindow struct {
	ID               string    `json:"id"`
	Label            string    `json:"label"`
	ScopeModel       string    `json:"scope_model,omitempty"`
	UsedPercent      float64   `json:"used_percent"`
	RemainingPercent float64   `json:"remaining_percent"`
	ResetAt          time.Time `json:"reset_at,omitempty,omitzero"`
	Known            bool      `json:"known"`
	HardExhausted    bool      `json:"hard_exhausted,omitempty"`
	Routing          bool      `json:"routing"`
	PeriodSeconds    int64     `json:"period_seconds,omitempty"`
}

func (s CapacityState) Clone() CapacityState {
	copyState := s
	if len(s.Windows) > 0 {
		copyState.Windows = append([]CapacityWindow(nil), s.Windows...)
	}
	return copyState
}

type capacityRank struct {
	known         bool
	exhausted     bool
	resetAt       time.Time
	urgencyKnown  bool
	urgency       float64
	budgetResetAt time.Time
}

func quotaCapacityRank(auth *Auth, model string, now time.Time) capacityRank {
	if auth == nil {
		return capacityRank{}
	}
	state := auth.Capacity
	if !state.Supported || state.FetchedAt.IsZero() {
		return capacityRank{}
	}
	if !state.StaleAt.IsZero() && !now.Before(state.StaleAt) {
		return capacityRank{}
	}
	rank := capacityRank{}
	for _, window := range state.Windows {
		if !window.Routing || !window.Known || !capacityScopeMatches(window.ScopeModel, model) {
			continue
		}
		if !window.ResetAt.IsZero() && !window.ResetAt.After(now) {
			continue
		}
		rank.known = true
		remaining := normalizedCapacityPercent(window.RemainingPercent)
		if window.HardExhausted || remaining <= quotaCapacityPercentEpsilon {
			rank.exhausted = true
			if resetBefore(window.ResetAt, rank.resetAt) {
				rank.resetAt = window.ResetAt
			}
		}
		period := time.Duration(window.PeriodSeconds) * time.Second
		if period < quotaBudgetWindowMinPeriod {
			continue
		}
		horizon := period
		if !window.ResetAt.IsZero() {
			horizon = window.ResetAt.Sub(now)
			if rank.budgetResetAt.IsZero() || window.ResetAt.Before(rank.budgetResetAt) {
				rank.budgetResetAt = window.ResetAt
			}
		}
		if horizon < quotaMinimumUrgencyHorizon {
			horizon = quotaMinimumUrgencyHorizon
		}
		urgency := remaining / horizon.Hours()
		if !rank.urgencyKnown || urgency > rank.urgency {
			rank.urgency = urgency
		}
		rank.urgencyKnown = true
	}
	return rank
}

func normalizedCapacityPercent(value float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0
	}
	return math.Max(0, math.Min(100, value))
}

func capacityScopeMatches(scopeModel, model string) bool {
	scopeModel = strings.ToLower(strings.TrimSpace(scopeModel))
	model = strings.ToLower(canonicalModelKey(model))
	if scopeModel == "" {
		return true
	}
	switch scopeModel {
	case quotaScopeAntigravityGemini:
		return strings.Contains(model, "gemini")
	case quotaScopeAntigravityClaudeGPT:
		return strings.Contains(model, "claude") || strings.Contains(model, "gpt") || strings.Contains(model, "oss")
	case quotaScopeXAIBuild:
		return model != "" && !strings.Contains(model, "image") && !strings.Contains(model, "imagine")
	case quotaScopeXAIImagine:
		return strings.Contains(model, "image") || strings.Contains(model, "imagine")
	default:
		return model != "" && strings.EqualFold(canonicalModelKey(scopeModel), model)
	}
}

func quotaCapacityExhausted(auth *Auth, model string, now time.Time) bool {
	rank := quotaCapacityRank(auth, model, now)
	return rank.known && rank.exhausted
}

func quotaRankLess(left, right capacityRank) bool {
	if left.urgencyKnown != right.urgencyKnown {
		return left.urgencyKnown
	}
	if !left.urgencyKnown {
		return false
	}
	return left.urgency-right.urgency > quotaCapacityUrgencyEpsilon
}

func resetBefore(left, right time.Time) bool {
	if left.IsZero() {
		return false
	}
	return right.IsZero() || left.Before(right)
}

func quotaDrainPriorities(available map[int][]*Auth) []int {
	priorities := make([]int, 0, len(available))
	for priority := range available {
		priorities = append(priorities, priority)
	}
	sort.Slice(priorities, func(i, j int) bool { return priorities[i] > priorities[j] })
	return priorities
}
