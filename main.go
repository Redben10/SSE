package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
)

type Tunnel struct {
	ID          string
	Content     string
	SubChannels map[string]string
}

var tunnels = make(map[string]*Tunnel)
var tunnelsMutex = &sync.Mutex{}
var clients = make(map[string]map[string][]chan string)
var clientsMutex = &sync.Mutex{}

func main() {
	log.Println("Starting server on port 2427")
	http.HandleFunc("/", withCORS(homePage))
	http.HandleFunc("/api/v2/tunnel/create", withCORS(createTunnel))
	http.HandleFunc("/api/v2/tunnel/stream", withCORS(streamTunnelContent))
	http.HandleFunc("/api/v2/tunnel/send", withCORS(sendToTunnel))
	log.Fatal(http.ListenAndServe(":2427", nil))
}

func homePage(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Welcome to the TXTTunnel homepage!")
}

func withCORS(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		handler(w, r)
	}
}

func createTunnel(w http.ResponseWriter, r *http.Request) {
	var requestData struct {
		ID string `json:"id"`
	}

	// Read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}

	// Unmarshal the JSON data into requestData
	err = json.Unmarshal(body, &requestData)
	if err != nil {
		http.Error(w, "Failed to parse JSON", http.StatusBadRequest)
		return
	}

	// Check if the ID is provided and not empty
	if requestData.ID == "" {
		http.Error(w, "Tunnel ID must be provided.", http.StatusBadRequest)
		return
	}

	// Check for uniqueness of the tunnel ID
	tunnelsMutex.Lock()
	_, exists := tunnels[requestData.ID]
	if exists {
		tunnelsMutex.Unlock()
		http.Error(w, "Tunnel ID already exists.", http.StatusConflict)
		return
	}

	// Create the tunnel
	tunnels[requestData.ID] = &Tunnel{ID: requestData.ID, Content: "", SubChannels: make(map[string]string)}
	tunnelsMutex.Unlock()

	// Respond with the created tunnel ID
	w.Header().Set("Content-Type", "application/json")
	response, err := json.Marshal(map[string]string{"id": requestData.ID})
	if err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
	w.Write(response)
	log.Printf("Tunnel %s has been created.", requestData.ID)
}

func sendToTunnel(w http.ResponseWriter, r *http.Request) {
	var requestData struct {
		ID         string `json:"id"`
		Content    string `json:"content"`
		SubChannel string `json:"subChannel"`
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}

	err = json.Unmarshal(body, &requestData)
	if err != nil {
		http.Error(w, "Failed to parse JSON", http.StatusBadRequest)
		return
	}

	if requestData.ID == "" {
		http.Error(w, "No tunnel id has been provided.", http.StatusBadRequest)
		return
	}

	if requestData.Content == "" {
		http.Error(w, "No content has been provided.", http.StatusBadRequest)
		return
	}

	if requestData.SubChannel == "" {
		http.Error(w, "No subChannel has been provided.", http.StatusBadRequest)
		return
	}

	tunnelsMutex.Lock()
	tunnel, exists := tunnels[requestData.ID]
	if !exists {
		tunnelsMutex.Unlock()
		http.Error(w, "No tunnel with this id exists.", http.StatusInternalServerError)
		return
	}
	tunnel.SubChannels[requestData.SubChannel] = requestData.Content
	tunnelsMutex.Unlock()

	clientsMutex.Lock()
	for _, client := range clients[requestData.ID][requestData.SubChannel] {
		client <- requestData.Content
	}
	clientsMutex.Unlock()

	log.Printf("Tunnel %s subChannel %s has been updated.", requestData.ID
