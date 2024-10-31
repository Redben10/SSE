package main

import (
    "fmt"
    "net/http"
    "github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool {
        return true // Allow all connections
    },
}

var clients = make(map[*websocket.Conn]bool) // Connected clients

func main() {
    http.HandleFunc("/chat", handleChat)
    http.HandleFunc("/", serveHome)
    fmt.Println("Server started on :8080")
    http.ListenAndServe(":8080", nil)
}

func serveHome(w http.ResponseWriter, r *http.Request) {
    http.ServeFile(w, r, "index.html")
}

func handleChat(w http.ResponseWriter, r *http.Request) {
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        fmt.Println("Error while upgrading connection:", err)
        return
    }
    defer conn.Close()

    clients[conn] = true // Add new client

    for {
        var msg string
        err := conn.ReadMessage(&msg)
        if err != nil {
            fmt.Println("Error while reading message:", err)
            delete(clients, conn) // Remove client on error
            break
        }
        broadcastMessage(msg) // Broadcast message to all clients
    }
}

func broadcastMessage(message string) {
    for client := range clients {
        err := client.WriteMessage(websocket.TextMessage, []byte(message))
        if err != nil {
            fmt.Println("Error while sending message:", err)
            client.Close()
            delete(clients, client) // Remove client on error
        }
    }
}
