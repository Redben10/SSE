package main

import (
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

    port := ":10000"
    fmt.Printf("Server is running on http://localhost%s\n", port)
    go broadcastMessages() // Start the message broadcasting goroutine
    log.Fatal(http.ListenAndServe(port, nil))
}

func handleSSE(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    w.Header().Set("Access-Control-Allow-Origin", "*")

    clientChan := make(chan string)
    clients[clientChan] = true
    defer delete(clients, clientChan)

    flusher, ok := w.(http.Flusher)
    if !ok {
        http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
        return
    }

    for {
        select {
        case msg := <-clientChan:
            fmt.Fprintf(w, "data: %s\n\n", msg)
            flusher.Flush()
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

    w.WriteHeader(http.StatusOK)
}

func broadcastMessages() {
    for msg := range messageChannel {
        for clientChan := range clients {
            select {
            case clientChan <- msg:
            default:
                // Client is not ready to receive, skip it
            }
        }
    }
}
