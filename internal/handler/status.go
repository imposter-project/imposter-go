package handler

import (
	"encoding/json"
	"net/http"

	"github.com/imposter-project/imposter-go/internal/version"
)

var (
	// cachedStatusResponse holds the pre-marshalled JSON response
	cachedStatusResponse []byte
)

func init() {
	// Initialise the cached response during package initialisation
	response := struct {
		Status  string `json:"status"`
		Version string `json:"version"`
	}{
		Status:  "ok",
		Version: version.Version,
	}

	// Marshal the response once
	var err error
	cachedStatusResponse, err = json.Marshal(response)
	if err != nil {
		// This should never happen with our simple struct,
		// but if it does, we want to panic early during startup
		panic("failed to marshal status response: " + err.Error())
	}
}

// handleStatusRequest handles the /system/status endpoint
func handleStatusRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(cachedStatusResponse)
}
