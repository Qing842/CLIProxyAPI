package auth

import "strings"

func (m *Manager) UpdateCapacity(authID string, state CapacityState) bool {
	if m == nil || strings.TrimSpace(authID) == "" {
		return false
	}
	state.Provider = strings.ToLower(strings.TrimSpace(state.Provider))
	state = state.Clone()
	m.mu.Lock()
	defer m.mu.Unlock()
	auth := m.auths[authID]
	if auth == nil {
		return false
	}
	auth.Capacity = state
	return true
}

func selectorUsesQuotaDrain(selector Selector) bool {
	switch typed := selector.(type) {
	case *QuotaDrainSelector:
		return true
	case *SessionAffinitySelector:
		return typed != nil && selectorUsesQuotaDrain(typed.fallback)
	default:
		return false
	}
}
