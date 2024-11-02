<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Chat Room</title>
</head>
<body>
    <h1>Welcome to TXTTunnel Chat</h1>
    
    <!-- New Username Section -->
    <div>
        <input type="text" id="usernameInput" placeholder="Enter your username" />
        <button id="setUsernameBtn">Set Username</button>
    </div>

    <h2>Current Room: <span id="currentRoom">global</span></h2>
    
    <button id="createRoomBtn">Create Room</button>
    <button id="joinRoomBtn">Join Room</button>
    
    <div id="chatMessages"></div>
    <input type="text" id="messageInput" placeholder="Type a message..." />
    <button id="sendMessageBtn">Send</button>

    <script>
        let username = ""; // To store the username

        document.getElementById("setUsernameBtn").onclick = function() {
            const input = document.getElementById("usernameInput").value;
            if (input) {
                username = input; // Set the username
                alert(`Username set to: ${username}`);
                document.getElementById("usernameInput").value = ""; // Clear input
            } else {
                alert("Please enter a username.");
            }
        };

        // Room creation functionality
        document.getElementById("createRoomBtn").onclick = function() {
            const roomId = prompt("Enter room ID to create:");
            if (roomId) {
                // Check if room exists and create it if it does not
                fetch('https://txttunnel.onrender.com/api/v2/tunnel/create', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ id: roomId })
                })
                .then(response => {
                    if (response.ok) {
                        alert("Room created successfully!");
                        // Update the current room
                        document.getElementById("currentRoom").innerText = roomId;
                    } else {
                        alert("This room is already created. Try joining it instead.");
                    }
                });
            }
        };

        // Room joining functionality
        document.getElementById("joinRoomBtn").onclick = function() {
            const roomId = prompt("Enter room ID to join:");
            if (roomId) {
                // Join the room (implement joining logic)
                alert(`Joining room: ${roomId}`);
                document.getElementById("currentRoom").innerText = roomId;
            }
        };

        // Message sending functionality
        document.getElementById("sendMessageBtn").onclick = function() {
            const message = document.getElementById("messageInput").value;
            if (message) {
                // Send the message along with the username
                const content = username ? `${username}: ${message}` : message;

                fetch('https://txttunnel.onrender.com/api/v2/tunnel/send', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                        id: document.getElementById("currentRoom").innerText,
                        content: content,
                        subChannel: "general" // Or whatever subChannel you want to use
                    })
                });

                document.getElementById("messageInput").value = ""; // Clear the input
            } else {
                alert("Please enter a message.");
            }
        };

        // Implement receiving messages logic to display them
    </script>
</body>
</html>
