package kamux

import (
	"sync"

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
		c.RemoveRequester()
		return "",err
	}
	return messageRecv,nil
}

// ReceiveJson receive json from ws and disconnect when stop receiving
func (c *WsContext) ReceiveJson() (map[string]any,error) {
	var data map[string]any
	err := websocket.JSON.Receive(c.Ws, &data)
	if err != nil {
		c.RemoveRequester()
		return nil,err
	}
	
	return data,nil
}

// Json send json to the client
func (c *WsContext) Json(data map[string]any) error {
	err := websocket.JSON.Send(c.Ws, data)
	if err != nil {
		c.RemoveRequester()
		return err
	}
	return nil
}

// Broadcast send message to all clients in c.Clients
func (c *WsContext) Broadcast(data any) error {
	m.RLock()
	for _,ws := range c.Route.Clients {
		err := websocket.JSON.Send(ws, data)
		if err != nil {
			c.RemoveRequester()
			m.RUnlock()
			return err
		}
	}
	m.RUnlock()
	return nil
}

// Broadcast send message to all clients in c.Clients
func (c *WsContext) BroadcastExceptCaller(data map[string]any) error {
	m.RLock()
	for _,ws := range c.Route.Clients {
		if ws != c.Ws {
			err := websocket.JSON.Send(ws, data)
			if err != nil {
				c.RemoveRequester()
				m.RUnlock()
				return err
			}
		}
	}
	m.RUnlock()
	return nil
}

// Text send text to the client
func (c *WsContext) Text(data string) error {
	err := websocket.Message.Send(c.Ws, data)
	if err != nil {
		c.RemoveRequester()
		return err
	}
	return nil
}

// RemoveRequester remove the client from Clients list in context
func (c *WsContext) RemoveRequester(name ...string) {
	m.Lock()
	for k, ws := range c.Route.Clients {
		if len(name) > 1 {
			n := name[0]
			if conn,ok := c.Route.Clients[n];ok {
				delete(c.Route.Clients,n)
				_ = conn.Close()
			}		
		} else {
			if ws == c.Ws {
				if conn,ok := c.Route.Clients[k];ok {
					delete(c.Route.Clients, k)
					_ = conn.Close()
				}
				
			}
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