package websockets

import (
	"sync"

	"github.com/gorilla/websocket"
)

type Client struct {
	Id       uint
	Conn     *websocket.Conn
	SendChan chan []byte
}

type WebSocketManager struct {
	clients map[uint]*Client
	mutex   sync.RWMutex
}

func (m *WebSocketManager) AddClient(userId uint, conn *websocket.Conn) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.clients[userId] = &Client{
		Id:       userId,
		Conn:     conn,
		SendChan: make(chan []byte, 256),
	}
}

func (m *WebSocketManager) RemoveClient(userId uint) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	delete(m.clients, userId)
}

func (m *WebSocketManager) GetClient(userId uint) (*Client, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	result, ok := m.clients[userId]
	return result, ok
}

var WSManager = &WebSocketManager{
	clients: make(map[uint]*Client),
}
