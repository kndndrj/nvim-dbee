package vim

import (
	"fmt"

	"github.com/kndndrj/nvim-dbee/dbee/conn"
)

type ConnectionStorage struct {
	connections map[string]*conn.Conn
}

func NewConnectionStorage() *ConnectionStorage {
	return &ConnectionStorage{
		connections: make(map[string]*conn.Conn),
	}
}

func (cs *ConnectionStorage) Register(c *conn.Conn) {
	cs.connections[c.GetID()] = c
}

func (cs *ConnectionStorage) Get(id string) (*conn.Conn, error) {
	c, ok := cs.connections[id]
	if !ok {
		return nil, fmt.Errorf("connection with id %q not registered", id)
	}
	return c, nil
}

func (cs *ConnectionStorage) Close() {
	for _, c := range cs.connections {
		c.Close()
	}
}
