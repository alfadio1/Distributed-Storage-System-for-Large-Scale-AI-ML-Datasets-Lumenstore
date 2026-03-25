package master

import (
	"log"
	"time"
)

const (
	heartbeatTimeout  = 5 * time.Second
	heartbeatScanTick = 1 * time.Second
)

// StartHeartbeatMonitor runs a background loop that marks nodes dead
// when they have not sent a heartbeat within heartbeatTimeout.
func (s *Server) StartHeartbeatMonitor() {
	ticker := time.NewTicker(heartbeatScanTick)

	go func() {
		defer ticker.Stop()

		for range ticker.C {
			s.mu.Lock()

			now := time.Now()
			for nodeID, node := range s.state.Nodes {
				aliveBefore := node.IsAlive
				aliveNow := now.Sub(node.LastSeen) <= heartbeatTimeout

				if aliveBefore != aliveNow {
					node.IsAlive = aliveNow
					s.state.Nodes[nodeID] = node

					if aliveNow {
						log.Printf("[master] node revived node=%s addr=%s", node.NodeID, node.Address)
					} else {
						log.Printf("[master] node marked dead node=%s addr=%s last_seen=%s",
							node.NodeID,
							node.Address,
							node.LastSeen.Format(time.RFC3339),
						)
					}
				}
			}

			s.mu.Unlock()
		}
	}()
}
