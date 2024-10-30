package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"sync"
)

type Tunnel struct {
	ID          string
	Content     string
	Messages    []string // Field to store messages
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
	http.HandleFunc("/api/v2/tunnel/messages", withCORS(getMessages)) // New endpoint for messages
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

func sendToTunnel(w http.ResponseWriter, r *http.Request) {
	var requestData struct {
		ID      string `json:"id"`
		Content string `json:"content"`
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

	tunnelsMutex.Lock()
	tunnel, exists := tunnels[requestData.ID]
	if !exists {
		tunnelsMutex.Unlock()
		http.Error(w, "No tunnel with this id exists.", http.StatusInternalServerError)
		return
	}

	// Add message to tunnel messages
	tunnel.Messages = append(tunnel.Messages, requestData.Content)
	tunnelsMutex.Unlock()

	clientsMutex.Lock()
	for _, client := range clients[requestData.ID]["default"] {
		client <- requestData.Content
	}
	clientsMutex.Unlock()

	log.Printf("Tunnel %s has been updated.", requestData.ID)
	fmt.Fprintf(w, "Tunnel %s has been updated.", requestData.ID)
}

func streamTunnelContent(w http.ResponseWriter, r *http.Request) {
	tunnelId := r.URL.Query().Get("id")
	if tunnelId == "" {
		http.Error(w, "No tunnel id has been provided.\nPlease use ?id= to include the tunnel id.", http.StatusBadRequest)
		return
	}

	tunnelsMutex.Lock()
	_, exists := tunnels[tunnelId]
	if !exists {
		tunnelsMutex.Unlock()
		http.Error(w, "No tunnel with this id exists.", http.StatusInternalServerError)
		return
	}
	tunnelsMutex.Unlock()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	clientChan := make(chan string)
	clientsMutex.Lock()
	if clients[tunnelId] == nil {
		clients[tunnelId] = make(map[string][]chan string)
	}
	clients[tunnelId]["default"] = append(clients[tunnelId]["default"], clientChan)
	clientsMutex.Unlock()

	for {
		select {
		case msg := <-clientChan:
			fmt.Fprintf(w, "data: %s\n\n", msg)
			w.(http.Flusher).Flush()
		case <-r.Context().Done():
			clientsMutex.Lock()
			for i, client := range clients[tunnelId]["default"] {
				if client == clientChan {
					clients[tunnelId]["default"] = append(clients[tunnelId]["default"][:i], clients[tunnelId]["default"][i+1:]...)
					break
				}
			}
			clientsMutex.Unlock()
			return
		}
	}
}

// Function to retrieve messages from a tunnel
func getMessages(w http.ResponseWriter, r *http.Request) {
	tunnelId := r.URL.Query().Get("id")
	if tunnelId == "" {
		http.Error(w, "No tunnel id has been provided.", http.StatusBadRequest)
		return
	}

	tunnelsMutex.Lock()
	tunnel, exists := tunnels[tunnelId]
	tunnelsMutex.Unlock()
	if !exists {
		http.Error(w, "No tunnel with this id exists.", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	response, err := json.Marshal(tunnel.Messages) // Send back the messages
	if err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
	w.Write(response)
}

func createTunnel(w http.ResponseWriter, r *http.Request) {
	var requestData struct {
		ID string `json:"id"`
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

	tunnelsMutex.Lock()
	if _, exists := tunnels[requestData.ID]; exists {
		tunnelsMutex.Unlock()
		http.Error(w, "Tunnel ID already exists.", http.StatusConflict)
		return
	}

	tunnels[requestData.ID] = &Tunnel{ID: requestData.ID, Content: "", Messages: []string{}, SubChannels: make(map[string]string)}
	tunnelsMutex.Unlock()

	w.Header().Set("Content-Type", "application/json")
	response, err := json.Marshal(map[string]string{"id": requestData.ID})
	if err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
	w.Write(response)
	log.Printf("Tunnel %s has been created.", requestData.ID)
}
