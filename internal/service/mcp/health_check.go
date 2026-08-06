package mcp

import (
	"context"
	"log"
	"time"

	"github.com/mcpjungle/mcpjungle/internal/model"
)

// HealthCheckConfig controls the periodic MCP server health check.
type HealthCheckConfig struct {
	// Interval between health check rounds.
	Interval time.Duration
	// MaxConsecutiveFailures before an online server is marked unhealthy.
	MaxConsecutiveFailures int
}

// DefaultHealthCheckConfig matches design doc §9.4.
var DefaultHealthCheckConfig = HealthCheckConfig{
	Interval:              60 * time.Second,
	MaxConsecutiveFailures: 3,
}

// StartHealthCheck runs a background loop that periodically checks the health of
// all online and unhealthy servers. Cancel ctx to stop.
func (m *MCPService) StartHealthCheck(ctx context.Context, cfg HealthCheckConfig) {
	ticker := time.NewTicker(cfg.Interval)
	defer ticker.Stop()
	log.Printf("[health] starting periodic health check (interval=%s, max_failures=%d)",
		cfg.Interval, cfg.MaxConsecutiveFailures)
	for {
		select {
		case <-ctx.Done():
			log.Println("[health] health check stopped")
			return
		case <-ticker.C:
			m.runHealthChecks(ctx, cfg)
		}
	}
}

func (m *MCPService) runHealthChecks(ctx context.Context, cfg HealthCheckConfig) {
	var servers []model.McpServer
	if err := m.db.Where("status IN ?",
		[]string{model.StatusOnline, model.StatusUnhealthy}).
		Find(&servers).Error; err != nil {
		log.Printf("[health] failed to load servers: %v", err)
		return
	}
	for _, srv := range servers {
		// Run each check in its own goroutine so one slow server doesn't block others.
		s := srv // capture loop variable
		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("[health] recovered from panic in health check for %s: %v", s.Name, r)
				}
			}()
			m.checkOneServer(ctx, s, cfg)
		}()
	}
}

// checkOneServer performs a lightweight MCP Initialize on the server (no
// ListTools) and records the outcome.
func (m *MCPService) checkOneServer(ctx context.Context, srv model.McpServer, cfg HealthCheckConfig) {
	checkCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// createMcpServerConnectionWithDB performs an MCP Initialize only.
	client, err := createMcpServerConnectionWithDB(checkCtx, m.db, &srv, 10, false)
	if err != nil {
		m.recordHealthFailure(srv, err, cfg)
		return
	}
	client.Close()
	m.recordHealthSuccess(srv)
}

func (m *MCPService) recordHealthFailure(srv model.McpServer, err error, cfg HealthCheckConfig) {
	newCount := srv.HealthFailCount + 1
	updates := map[string]interface{}{
		"health_fail_count":    newCount,
		"last_health_check_at": time.Now().UTC(),
		"last_error_summary":   truncateErr(err.Error(), 500),
	}
	if srv.Status == model.StatusOnline && newCount >= cfg.MaxConsecutiveFailures {
		updates["status"] = model.StatusUnhealthy
		log.Printf("[health] server %s → unhealthy (failures=%d): %v", srv.Name, newCount, err)
	}
	_ = m.db.Model(&model.McpServer{}).Where("id = ?", srv.ID).Updates(updates).Error
}

func (m *MCPService) recordHealthSuccess(srv model.McpServer) {
	updates := map[string]interface{}{
		"health_fail_count":    0,
		"last_health_check_at": time.Now().UTC(),
		"last_error_summary":   "",
		"last_validated_at":    time.Now().UTC(),
	}
	if srv.Status == model.StatusUnhealthy {
		updates["status"] = model.StatusPendingPublish
		// Tools remain registered in the proxy; re-publish is required to go back online.
		log.Printf("[health] server %s → pending_publish (recovered)", srv.Name)
	}
	_ = m.db.Model(&model.McpServer{}).Where("id = ?", srv.ID).Updates(updates).Error
}

func truncateErr(s string, max int) string {
	if len(s) > max {
		return s[:max]
	}
	return s
}
