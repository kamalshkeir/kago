package kamux

import (
	"sync"

	"github.com/kamalshkeir/kago/core/utils/logger"

	"golang.org/x/net/websocket"
)


var m sync.RWMutex

type WsContext struct {
	Ws      *websocket.Conn
	Params  map[string]string
	Route
}

// ReceiveText receive text from ws and disconnect when stop receiving
func (c *WsContext) ReceiveText() (string,error) {
	var messageRecv string
	err := websocket.Message.Receive(c.Ws, &messageRecv)
	if err != nil {
		cerr := c.Ws.Close()
		if cerr != nil {
			return "",cerr
		}
		c.deleteRequesterClient()
		return "",err
	}
	return messageRecv,nil
}

// ReceiveJson receive json from ws and disconnect when stop receiving
func (c *WsContext) ReceiveJson() (map[string]interface{},error) {
	var data map[string]interface{}
	err := websocket.JSON.Receive(c.Ws, &data)
	if err != nil {
		cerr := c.Ws.Close()
		if cerr != nil {
			return nil,cerr
		}
		c.deleteRequesterClient()
		return nil,err
	}
	
	return data,nil
}

// Json send json to the client
func (c *WsContext) Json(data map[string]interface{}) {
	err := websocket.JSON.Send(c.Ws, data)
	logger.CheckError(err)
}


// Broadcast send message to all clients in c.Clients
func (c *WsContext) Broadcast(data interface{}) {
	m.RLock()
	for _,ws := range c.Route.Clients {
		err := websocket.JSON.Send(ws, data)
		logger.CheckError(err)
	}
	m.RUnlock()
}

// Broadcast send message to all clients in c.Clients
func (c *WsContext) BroadcastExceptCaller(data map[string]interface{}) {
	m.RLock()
	for _,ws := range c.Route.Clients {
		if ws != c.Ws {
			err := websocket.JSON.Send(ws, data)
			logger.CheckError(err)
		}
	}
	m.RUnlock()
}

// Text send text to the client
func (c *WsContext) Text(data string) {
	err := websocket.Message.Send(c.Ws, data)
	logger.CheckError(err)
}

// RemoveClient remove the client from Clients list in context
func (c *WsContext) deleteRequesterClient() {
	m.Lock()
	for k, ws := range c.Route.Clients {
		if ws == c.Ws {
			delete(c.Route.Clients, k)
		}
	}
	m.Unlock()
}

// AddClient add client to clients_list
func (c *WsContext) AddClient(key string) {
	m.Lock()
	if _,ok := c.Route.Clients[key];!ok {
		c.Route.Clients[key]=c.Ws
	}
	m.Unlock()
}