package handler

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/imposter-project/imposter-go/internal/config"
)

const (
	defaultMaxAge = 86400 // 24 hours
)

// handleCORS processes CORS headers and preflight requests
func handleCORS(w http.ResponseWriter, r *http.Request, corsConfig *config.CorsConfig) bool {
	if corsConfig == nil {
		return false
	}

	origin := r.Header.Get("Origin")
	if origin == "" {
		// For preflight requests without Origin, return 400
		if r.Method == http.MethodOptions {
			log.Printf("Warning: Preflight request received without Origin header")
			w.WriteHeader(http.StatusBadRequest)
			return true
		}
		// For regular requests without Origin, do nothing
		return false
	}

	// Handle preflight requests
	if r.Method == http.MethodOptions {
		handlePreflightRequest(w, r, corsConfig)
		return true
	}

	// Add CORS headers to actual response
	addCORSHeaders(w, r, corsConfig)
	return false
}

// handlePreflightRequest handles CORS preflight requests
func handlePreflightRequest(w http.ResponseWriter, r *http.Request, corsConfig *config.CorsConfig) {
	addCORSHeaders(w, r, corsConfig)

	// Add preflight-specific headers
	if len(corsConfig.AllowMethods) > 0 {
		w.Header().Set("Access-Control-Allow-Methods", strings.Join(corsConfig.AllowMethods, ", "))
	} else {
		// Default to common methods if none specified
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
	}

	if len(corsConfig.AllowHeaders) > 0 {
		w.Header().Set("Access-Control-Allow-Headers", strings.Join(corsConfig.AllowHeaders, ", "))
	} else {
		// Echo requested headers if none specified
		if requestedHeaders := r.Header.Get("Access-Control-Request-Headers"); requestedHeaders != "" {
			w.Header().Set("Access-Control-Allow-Headers", requestedHeaders)
		}
	}

	// Set max age for preflight response caching
	maxAge := defaultMaxAge
	if corsConfig.MaxAge > 0 {
		maxAge = corsConfig.MaxAge
	}
	w.Header().Set("Access-Control-Max-Age", strconv.Itoa(maxAge))

	w.WriteHeader(http.StatusNoContent)
}

// addCORSHeaders adds CORS headers to the response
func addCORSHeaders(w http.ResponseWriter, r *http.Request, corsConfig *config.CorsConfig) {
	origin := r.Header.Get("Origin")
	allowedOrigins := corsConfig.GetAllowedOrigins()

	// Handle Allow-Origin
	if containsString(allowedOrigins, "all") {
		// Echo the origin
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Vary", "Origin")
	} else if containsString(allowedOrigins, "*") {
		w.Header().Set("Access-Control-Allow-Origin", "*")
	} else if len(allowedOrigins) > 0 && containsString(allowedOrigins, origin) {
		// Only set header if origin is in allowed list
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Vary", "Origin")
	}

	// Handle credentials
	if corsConfig.AllowCredentials {
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	}
}

// containsString checks if a string slice contains a specific string
func containsString(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}
