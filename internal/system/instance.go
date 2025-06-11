package system

import (
	"fmt"
	"os"
	"time"
)

// GenerateInstanceID generates a unique instance ID for this server instance
func GenerateInstanceID() string {
	hostname, _ := os.Hostname()
	pid := os.Getpid()
	timestamp := time.Now().UnixNano()
	return fmt.Sprintf("%s-%d-%d", hostname, pid, timestamp)
}
