package handler

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"log"

	"github.com/gatehill/imposter-go/internal/store"
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
	if key == "" {
		query := r.URL.Query().Get("keyPrefix")
		items := store.GetAllValues(storeName, query)
		log.Printf("Listing all items in store: %s", storeName)
		if err := json.NewEncoder(w).Encode(items); err != nil {
			log.Printf("Failed to encode items: %v", err)
			http.Error(w, "Failed to encode items", http.StatusInternalServerError)
		}
	} else {
		value, found := store.GetValue(storeName, key)
		if !found {
			log.Printf("Item not found: %s in store: %s", key, storeName)
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		log.Printf("Returning item: %s from store: %s", key, storeName)
		if strVal, ok := value.(string); ok {
			fmt.Fprint(w, strVal)
		} else {
			if err := json.NewEncoder(w).Encode(value); err != nil {
				log.Printf("Failed to encode value: %v", err)
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
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("Failed to read request body: %v", err)
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}
	store.StoreValue(storeName, key, string(body))
	log.Printf("Saved item: %s to store: %s", key, storeName)
	w.WriteHeader(http.StatusNoContent)
}

func handlePostStore(w http.ResponseWriter, r *http.Request, storeName string) {
	var items map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&items); err != nil {
		log.Printf("Invalid JSON: %v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	for key, value := range items {
		store.StoreValue(storeName, key, value)
	}
	log.Printf("Saved %d items to store: %s", len(items), storeName)
	w.WriteHeader(http.StatusNoContent)
}

func handleDeleteStore(w http.ResponseWriter, storeName, key string) {
	if key == "" {
		store.DeleteStore(storeName)
		log.Printf("Deleted store: %s", storeName)
	} else {
		store.DeleteValue(storeName, key)
		log.Printf("Deleted item: %s from store: %s", key, storeName)
	}
	w.WriteHeader(http.StatusNoContent)
}
