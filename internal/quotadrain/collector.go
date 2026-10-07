package quotadrain

import (
	"context"
	"strings"
	"sync"
	"time"

	coreauth "github.com/router-for-me/CLIProxyAPI/v8/sdk/cliproxy/auth"
	log "github.com/sirupsen/logrus"
)

const (
	refreshInterval = 5 * time.Minute
	staleAfter      = 10 * time.Minute
	maxWorkers      = 4
)

func StrategyEnabled(strategy string) bool {
	switch strings.ToLower(strings.TrimSpace(strategy)) {
	case "quota-drain", "quotadrain", "qd":
		return true
	default:
		return false
	}
}

type Collector struct {
	manager *coreauth.Manager
	mu      sync.RWMutex
	enabled bool
	cancel  context.CancelFunc
	wake    chan struct{}
}

func NewCollector(manager *coreauth.Manager) *Collector {
	return &Collector{manager: manager, wake: make(chan struct{}, 1)}
}

func (c *Collector) Start(parent context.Context, enabled bool) {
	if c == nil || c.manager == nil {
		return
	}
	if parent == nil {
		parent = context.Background()
	}
	c.mu.Lock()
	c.enabled = enabled
	if c.cancel != nil {
		c.mu.Unlock()
		c.Wake()
		return
	}
	ctx, cancel := context.WithCancel(parent)
	c.cancel = cancel
	c.mu.Unlock()
	go c.run(ctx)
}

func (c *Collector) UpdateEnabled(enabled bool) {
	if c == nil {
		return
	}
	c.mu.Lock()
	changed := c.enabled != enabled
	c.enabled = enabled
	c.mu.Unlock()
	if changed && enabled {
		c.Wake()
	}
}

func (c *Collector) Wake() {
	if c == nil {
		return
	}
	select {
	case c.wake <- struct{}{}:
	default:
	}
}

func (c *Collector) Stop() {
	if c == nil {
		return
	}
	c.mu.Lock()
	cancel := c.cancel
	c.cancel = nil
	c.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}

func (c *Collector) isEnabled() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.enabled
}

func (c *Collector) run(ctx context.Context) {
	timer := time.NewTimer(0)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
		case <-c.wake:
		}
		if c.isEnabled() {
			c.collect(ctx)
		}
		if ctx.Err() != nil {
			return
		}
		resetTimer(timer, refreshInterval)
	}
}

func resetTimer(timer *time.Timer, interval time.Duration) {
	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
	timer.Reset(interval)
}

func (c *Collector) collect(ctx context.Context) {
	auths := c.manager.List()
	jobs := make(chan *coreauth.Auth)
	workers := maxWorkers
	if len(auths) < workers {
		workers = len(auths)
	}
	if workers == 0 {
		return
	}
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for auth := range jobs {
				c.collectAuth(ctx, auth)
			}
		}()
	}
	for _, auth := range auths {
		if !supportedAuth(auth) {
			continue
		}
		select {
		case <-ctx.Done():
			close(jobs)
			wg.Wait()
			return
		case jobs <- auth:
		}
	}
	close(jobs)
	wg.Wait()
}

func supportedAuth(auth *coreauth.Auth) bool {
	if auth == nil || auth.Disabled || auth.Status == coreauth.StatusDisabled || auth.AuthKind() != coreauth.AuthKindOAuth {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(auth.Provider)) {
	case "antigravity", "codex", "xai":
		return true
	default:
		return false
	}
}

func (c *Collector) collectAuth(ctx context.Context, auth *coreauth.Auth) {
	if auth == nil || ctx.Err() != nil {
		return
	}
	provider := strings.ToLower(strings.TrimSpace(auth.Provider))
	now := time.Now().UTC()
	windows, errFetch := fetchProviderCapacity(ctx, c.manager, auth, provider, now)
	state := auth.Capacity
	state.Provider = provider
	state.Supported = true
	state.LastAttemptAt = now
	if errFetch != nil {
		state.LastError = safeError(errFetch)
		c.manager.UpdateCapacity(auth.ID, state)
		if ctx.Err() == nil {
			log.WithFields(log.Fields{"auth_id": auth.ID, "provider": provider}).Debug("quota-drain refresh failed")
		}
		return
	}
	state.FetchedAt = now
	state.StaleAt = now.Add(staleAfter)
	state.LastError = ""
	state.Windows = windows
	c.manager.UpdateCapacity(auth.ID, state)
}

func safeError(err error) string {
	if err == nil {
		return ""
	}
	message := strings.TrimSpace(err.Error())
	if len(message) > 240 {
		message = message[:240]
	}
	return message
}
