package utils

import (
	"fmt"
)

// LogInfo logs an informational message
func LogInfo(msg string) {
	fmt.Println("[INFO]:", msg)
}
