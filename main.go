package main

import (
    "encoding/json"
    "fmt"
    "io"
    "log"
    "net/http"
    "sync"
    "time"
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

const globalRoomId = "global"

func main() {
    // Create the global room on startup
    tunnelsMutex.Lock()
    tunnels[globalRoomId] = &Tunnel{ID: globalRoomId, Content: "", SubChannels: make(map[string]string)}
    tunnelsMutex.Unlock()

    // Start the ping goroutine
    go pingGlobalRoom()

    log.Println("Starting server on port 2427")
    http.HandleFunc("/", withCORS(homePage))
    http.HandleFunc("/api/v2/tunnel/create", withCORS(createTunnel))
    http.HandleFunc("/api/v2/tunnel/checkRoomExists", withCORS(checkRoomExists))
    http.HandleFunc("/api/v2/tunnel/stream", withCORS(streamTunnelContent))
    http.HandleFunc("/api/v2/tunnel/send", withCORS(sendToTunnel))
    log.Fatal(http.ListenAndServe(":2427", nil))
}

// Function to send a ping message to the global room every 10 minutes
func pingGlobalRoom() {
    ticker := time.NewTicker(10 * time.Minute)
    defer ticker.Stop()

    for range ticker.C {
        tunnelsMutex.Lock()
        tunnel, exists := tunnels[globalRoomId]
        if exists {
            tunnel.SubChannels["main"] = "Ping from server"
            log.Println("Sent ping to global room")

            clientsMutex.Lock()
            for _, client := range clients[globalRoomId]["main"] {
                client <- "Ping from server"
            }
            clientsMutex.Unlock()
        }
        tunnelsMutex.Unlock()
    }
}

// (rest of the existing code)
