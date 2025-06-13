# AI Chat Application Architecture

## Overview

The AI Chat application is a web-based chat interface that allows users to interact with an AI assistant. The application is built using Go for the backend and HTML/CSS/JavaScript for the frontend. It uses WebSockets for real-time communication between the client and server.

## System Architecture

The application follows a client-server architecture with the following high-level components:

```
┌─────────────┐      HTTP/WebSocket      ┌─────────────────────────────────────┐
│             │◄─────────────────────────┤                                     │
│  Web Client │                          │           Go Web Server             │
│             │─────────────────────────►│                                     │
└─────────────┘                          └─────────────────────────────────────┘
                                                          │
                                                          │
                                                          ▼
                                         ┌─────────────────────────────────────┐
                                         │                                     │
                                         │         Session Management          │
                                         │                                     │
                                         └─────────────────────────────────────┘
                                                          │
                                                          │
                                                          ▼
                                         ┌─────────────────────────────────────┐
                                         │                                     │
                                         │          Chat Processing            │
                                         │                                     │
                                         └─────────────────────────────────────┘
```

## Component Details

### Backend Components

1. **HTTP Server (main.go)**
   - Initializes and configures the web server
   - Sets up routes and middleware
   - Manages server lifecycle

2. **HTTP Handlers (httpHandlers)**
   - Processes HTTP requests
   - Renders templates
   - Manages user sessions via cookies
   - Handles chat message submission

3. **WebSocket Server (websocketServer)**
   - Manages WebSocket connections
   - Publishes messages to specific clients
   - Enables real-time updates

4. **Session Manager (sessions)**
   - Manages user sessions
   - Maps session IDs to chat sessions
   - Handles session creation and cleanup

5. **Chat Session (chatSession)**
   - Represents a conversation between a user and the AI
   - Manages chat blocks (user-assistant message pairs)
   - Processes user messages and generates AI responses

6. **Configuration (config)**
   - Manages application configuration
   - Handles environment variables and command-line flags

7. **Static Assets (staticAssets)**
   - Serves static files (HTML, CSS, JavaScript)
   - Embeds frontend assets into the binary

### Frontend Components

1. **Templates (web/templates)**
   - Go HTML templates for rendering the UI
   - Includes main page and chat response templates

2. **Frontend Assets (web/frontend)**
   - CSS for styling
   - JavaScript for client-side functionality
   - Static assets like images

## Data Flow

1. **User Interaction Flow**
   - User accesses the application via web browser
   - Server creates a new session if one doesn't exist
   - User sends a message through the web interface
   - Message is sent to the server via HTTP POST
   - Server processes the message and generates a response
   - Response is sent back to the client via WebSocket
   - Client updates the UI with the response

2. **Session Management Flow**
   - User is assigned a UUID stored in a cookie
   - UUID is used to identify the user's session
   - Session manager maintains a map of active sessions
   - When a user disconnects or times out, the session is cleaned up

## Technologies Used

1. **Backend**
   - Go (Golang) - Primary programming language
   - Chi - HTTP router
   - Zerolog - Logging
   - HTML Templates - Server-side rendering
   - WebSockets - Real-time communication

2. **Frontend**
   - HTML/CSS - Structure and styling
   - JavaScript - Client-side interactivity
   - Embedded assets - Packaging static files with the binary

## Deployment Considerations

1. **Configuration**
   - The application can be configured via environment variables or command-line flags
   - Key configuration options include:
     - Host interface (default: localhost)
     - Port (default: 8080)
     - Simulated delay for testing (default: 0ms)

2. **Scaling**
   - The application uses in-memory session storage, which may need to be replaced with a distributed solution for high-scale deployments
   - WebSocket connections require consideration for load balancing

3. **Security**
   - Sessions are managed via cookies
   - CORS is configured to allow connections from HTTP and HTTPS origins

## Future Enhancements

Based on the TODO items in the README:

1. **Tools Integration** - Integration with external tools and services
2. **Session Management Improvements**:
   - Session type locking
   - Session timeout and scheduled cleanup
   - Improved WebSocket behavior after timeout
3. **UI Enhancements**:
   - Incremental updates from server
   - Disable controls during server response
   - Cancel answer button
   - "Answering" UI spinner
   - Better text formatting for user input