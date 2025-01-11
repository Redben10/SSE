package main

import (
    "bufio"
    "fmt"
    "log"
    "net/http"
    "os/exec"
    "strings"
)

var (
    messageChannel = make(chan string)
    clients        = make(map[chan string]bool)
)

func main() {
    http.HandleFunc("/events", handleSSE)
    http.HandleFunc("/send", handleSend)
    http.Handle("/", http.FileServer(http.Dir("."))) // Serve static files

    fmt.Println("Server is running on http://localhost:8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleSSE(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    w.Header().Set("Access-Control-Allow-Origin", "*")

    clientChan := make(chan string)
    clients[clientChan] = true
    defer delete(clients, clientChan)

    for {
        select {
        case msg := <-clientChan:
            fmt.Fprintf(w, "data: %s\n\n", msg)
            w.(http.Flusher).Flush()
        case <-r.Context().Done():
            return
        }
    }
}

func handleSend(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }

    message := r.FormValue("message")
    if message == "" {
        http.Error(w, "Message is required", http.StatusBadRequest)
        return
    }

    // Run Python script
    cmd := exec.Command("python", "ai_script.py", message)
    output, err := cmd.Output()
    if err != nil {
        http.Error(w, "Error running Python script", http.StatusInternalServerError)
        return
    }

    response := strings.TrimSpace(string(output))
    messageChannel <- response

    for clientChan := range clients {
        select {
        case clientChan <- response:
        default:
            // Client is not ready to receive, skip it
        }
    }

    w.WriteHeader(http.StatusOK)
}
