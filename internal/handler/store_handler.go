package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/imposter-project/imposter-go/internal/store"
	"github.com/imposter-project/imposter-go/pkg/logger"
)

// HandleStoreRequest handles requests to the /system/store API.
func HandleStoreRequest(w http.ResponseWriter, r *http.Request) {
	pathSegments := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(pathSegments) < 3 {
		http.Error(w, "Invalid store path", http.StatusBadRequest)
		return
	}

	storeName := pathSegments[2]
	key := ""
	if len(pathSegments) > 3 {
		key = strings.Join(pathSegments[3:], "/")
	}

	switch r.Method {
	case http.MethodGet:
		handleGetStore(w, r, storeName, key)
	case http.MethodPut:
		handlePutStore(w, r, storeName, key)
	case http.MethodPost:
		handlePostStore(w, r, storeName)
	case http.MethodDelete:
		handleDeleteStore(w, storeName, key)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleGetStore(w http.ResponseWriter, r *http.Request, storeName, key string) {
	s := store.Open(storeName, nil)
	if key == "" {
		query := r.URL.Query().Get("keyPrefix")
		items := s.GetAllValues(query)
		logger.Infof("Listing all items in store: %s", storeName)
		if err := json.NewEncoder(w).Encode(items); err != nil {
			logger.Errorf("Failed to encode items: %v", err)
			http.Error(w, "Failed to encode items", http.StatusInternalServerError)
		}
	} else {
		value, found := s.GetValue(key)
		if !found {
			logger.Infof("Item not found: %s in store: %s", key, storeName)
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		logger.Infof("Returning item: %s from store: %s", key, storeName)
		if strVal, ok := value.(string); ok {
			fmt.Fprint(w, strVal)
		} else {
			if err := json.NewEncoder(w).Encode(value); err != nil {
				logger.Errorf("Failed to encode value: %v", err)
				http.Error(w, "Failed to encode value", http.StatusInternalServerError)
			}
		}
	}
}

func handlePutStore(w http.ResponseWriter, r *http.Request, storeName, key string) {
	if key == "" {
		http.Error(w, "Key is required", http.StatusBadRequest)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Errorf("Failed to read request body: %v", err)
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}
	s := store.Open(storeName, nil)
	s.StoreValue(key, string(body))
	logger.Infof("Saved item: %s to store: %s", key, storeName)
	w.WriteHeader(http.StatusNoContent)
}

func handlePostStore(w http.ResponseWriter, r *http.Request, storeName string) {
	var items map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&items); err != nil {
		logger.Errorf("Invalid JSON: %v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	s := store.Open(storeName, nil)
	for key, value := range items {
		s.StoreValue(key, value)
	}
	logger.Infof("Saved %d items to store: %s", len(items), storeName)
	w.WriteHeader(http.StatusNoContent)
}

func handleDeleteStore(w http.ResponseWriter, storeName, key string) {
	if key == "" {
		store.DeleteStore(storeName)
		logger.Infof("Deleted store: %s", storeName)
	} else {
		s := store.Open(storeName, nil)
		s.DeleteValue(key)
		logger.Infof("Deleted item: %s from store: %s", key, storeName)
	}
	w.WriteHeader(http.StatusNoContent)
}
