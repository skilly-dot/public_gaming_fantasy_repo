package websocket

import (
    "encoding/json"
    "log"
    "sync"
    "time"

    "github.com/gorilla/websocket"
)

type Client struct {
    Hub      *Hub
    Conn     *websocket.Conn
    Send     chan []byte
    UserID   string
    LeagueID string
}

type Hub struct {
    Clients    map[*Client]bool
    Broadcast  chan []byte
    Register   chan *Client
    Unregister chan *Client
    mu         sync.RWMutex
}

type Message struct {
    Type     string      `json:"type"`
    Data     interface{} `json:"data"`
    UserID   string      `json:"user_id,omitempty"`
    LeagueID string      `json:"league_id,omitempty"`
}

func NewHub() *Hub {
    return &Hub{
        Clients:    make(map[*Client]bool),
        Broadcast:  make(chan []byte),
        Register:   make(chan *Client),
        Unregister: make(chan *Client),
    }
}

func (h *Hub) Run() {
    for {
        select {
        case client := <-h.Register:
            h.mu.Lock()
            h.Clients[client] = true
            h.mu.Unlock()

        case client := <-h.Unregister:
            h.mu.Lock()
            if _, ok := h.Clients[client]; ok {
                delete(h.Clients, client)
                close(client.Send)
            }
            h.mu.Unlock()

        case message := <-h.Broadcast:
            h.mu.RLock()
            for client := range h.Clients {
                select {
                case client.Send <- message:
                default:
                    close(client.Send)
                    delete(h.Clients, client)
                }
            }
            h.mu.RUnlock()
        }
    }
}

// SendToUser sends a message to a specific user
func (h *Hub) SendToUser(userID string, messageType string, data interface{}) {
    msg := Message{
        Type:   messageType,
        Data:   data,
        UserID: userID,
    }

    bytes, err := json.Marshal(msg)
    if err != nil {
        log.Printf("WebSocket marshal error: %v", err)
        return
    }

    h.mu.RLock()
    for client := range h.Clients {
        if client.UserID == userID {
            select {
            case client.Send <- bytes:
            default:
                close(client.Send)
                delete(h.Clients, client)
            }
        }
    }
    h.mu.RUnlock()
}

// SendToLeague sends to all users in a specific league
func (h *Hub) SendToLeague(leagueID string, messageType string, data interface{}) {
    msg := Message{
        Type:     messageType,
        Data:     data,
        LeagueID: leagueID,
    }

    bytes, err := json.Marshal(msg)
    if err != nil {
        log.Printf("WebSocket marshal error: %v", err)
        return
    }

    h.mu.RLock()
    for client := range h.Clients {
        if client.LeagueID == leagueID {
            select {
            case client.Send <- bytes:
            default:
                close(client.Send)
                delete(h.Clients, client)
            }
        }
    }
    h.mu.RUnlock()
}

// SendToAll broadcasts to all connected clients
func (h *Hub) SendToAll(messageType string, data interface{}) {
    msg := Message{
        Type: messageType,
        Data: data,
    }

    bytes, err := json.Marshal(msg)
    if err != nil {
        log.Printf("WebSocket marshal error: %v", err)
        return
    }

    h.Broadcast <- bytes
}

// WritePump writes messages to the WebSocket connection
func (c *Client) WritePump() {
    ticker := time.NewTicker(30 * time.Second)
    defer func() {
        ticker.Stop()
        c.Conn.Close()
    }()

    for {
        select {
        case message, ok := <-c.Send:
            c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
            if !ok {
                c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
                return
            }

            w, err := c.Conn.NextWriter(websocket.TextMessage)
            if err != nil {
                return
            }
            w.Write(message)

            // Add queued messages
            n := len(c.Send)
            for i := 0; i < n; i++ {
                w.Write([]byte{'\n'})
                w.Write(<-c.Send)
            }

            if err := w.Close(); err != nil {
                return
            }

        case <-ticker.C:
            c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
            if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
                return
            }
        }
    }
}

// ReadPump reads messages from the WebSocket connection
func (c *Client) ReadPump() {
    defer func() {
        c.Hub.Unregister <- c
        c.Conn.Close()
    }()

    c.Conn.SetReadLimit(512)
    c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
    c.Conn.SetPongHandler(func(string) error {
        c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
        return nil
    })

    for {
        _, _, err := c.Conn.ReadMessage()
        if err != nil {
            if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
                log.Printf("WebSocket error: %v", err)
            }
            break
        }
    }
}