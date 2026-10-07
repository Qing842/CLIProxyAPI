package auth

import "time"

const quotaBudgetRolloverTolerance = time.Hour

type quotaPinCandidate struct {
	id      string
	rank    capacityRank
	allowed bool
}

type quotaDrainPin struct {
	authID  string
	members map[string]time.Time
}

func (p *quotaDrainPin) validFor(candidates []quotaPinCandidate) (int, bool) {
	if p == nil || p.authID == "" {
		return -1, false
	}
	pinned := -1
	for index, candidate := range candidates {
		recorded, ok := p.members[candidate.id]
		if !ok || budgetRolledOver(recorded, candidate.rank.budgetResetAt) {
			return -1, false
		}
		if candidate.id == p.authID {
			if !candidate.rank.urgencyKnown {
				return -1, false
			}
			pinned = index
		}
	}
	return pinned, pinned >= 0
}

func (p *quotaDrainPin) set(authID string, candidates []quotaPinCandidate) {
	if p == nil {
		return
	}
	p.authID = authID
	p.members = make(map[string]time.Time, len(candidates))
	for _, candidate := range candidates {
		p.members[candidate.id] = candidate.rank.budgetResetAt
	}
}

func budgetRolledOver(recorded, current time.Time) bool {
	if recorded.IsZero() != current.IsZero() {
		return true
	}
	diff := current.Sub(recorded)
	if diff < 0 {
		diff = -diff
	}
	return diff > quotaBudgetRolloverTolerance
}

func pickQuotaPinned(pin *quotaDrainPin, candidates []quotaPinCandidate, order []int) int {
	if len(candidates) == 0 {
		return -1
	}
	if pinned, ok := pin.validFor(candidates); ok {
		if candidates[pinned].allowed {
			return pinned
		}
		return bestQuotaCandidate(candidates, order, true)
	}
	best := bestQuotaCandidate(candidates, order, false)
	if best >= 0 && pin != nil && candidates[best].rank.urgencyKnown {
		pin.set(candidates[best].id, candidates)
	}
	if best >= 0 && candidates[best].allowed {
		return best
	}
	return bestQuotaCandidate(candidates, order, true)
}

func bestQuotaCandidate(candidates []quotaPinCandidate, order []int, allowedOnly bool) int {
	best := -1
	for _, index := range order {
		candidate := candidates[index]
		if allowedOnly && !candidate.allowed {
			continue
		}
		if best < 0 || quotaRankLess(candidate.rank, candidates[best].rank) {
			best = index
		}
	}
	return best
}
