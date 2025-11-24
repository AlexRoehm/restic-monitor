package api

import (
	"context"
	"log"
	"time"
)

// markStaleAgentsOffline marks agents as offline if they haven't sent a heartbeat within the configured timeout
// Returns the number of agents marked offline
func (a *API) markStaleAgentsOffline() int {
	threshold := time.Now().Add(-time.Duration(a.config.HeartbeatTimeoutSeconds) * time.Second)

	// Update agents that are currently online but haven't been seen since threshold
	// Also mark agents with nil LastSeenAt as offline
	result := a.store.GetDB().Exec(`
		UPDATE agents 
		SET status = ?, updated_at = ?
		WHERE tenant_id = ? 
		  AND status = ?
		  AND (last_seen_at IS NULL OR last_seen_at < ?)
	`, "offline", time.Now(), a.store.GetTenantID(), "online", threshold)

	if result.Error != nil {
		log.Printf("Error marking stale agents offline: %v", result.Error)
		return 0
	}

	affectedRows := int(result.RowsAffected)
	if affectedRows > 0 {
		log.Printf("Marked %d agents offline due to heartbeat timeout (threshold: %d seconds)",
			affectedRows, a.config.HeartbeatTimeoutSeconds)
	}

	return affectedRows
}

// StartAgentMonitor starts a background goroutine that periodically checks for stale agents
// and marks them offline. Should be called once during API initialization.
func (a *API) StartAgentMonitor(ctx context.Context) {
	if a.config.HeartbeatTimeoutSeconds <= 0 {
		log.Println("Agent monitoring disabled (HeartbeatTimeoutSeconds <= 0)")
		return
	}

	// Check every minute
	ticker := time.NewTicker(60 * time.Second)

	log.Printf("Agent monitoring started (checking every 60s, timeout: %ds)", a.config.HeartbeatTimeoutSeconds)

	go func() {
		defer ticker.Stop()

		// Run once immediately
		a.markStaleAgentsOffline()

		for {
			select {
			case <-ctx.Done():
				log.Println("Agent monitoring stopped")
				return
			case <-ticker.C:
				a.markStaleAgentsOffline()
			}
		}
	}()
}
