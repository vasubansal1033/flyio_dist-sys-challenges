package main

import (
	"encoding/json"
	"log"

	"github.com/google/uuid"
	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

type Server struct {
	Node    *maelstrom.Node
	Counter int
}

func (s *Server) HandleGenerate(msg maelstrom.Message) error {
	var body map[string]any

	if err := json.Unmarshal(msg.Body, &body); err != nil {
		return err
	}

	body["type"] = "generate_ok"
	body["id"] = uuid.New().String()

	return s.Node.Reply(msg, body)
}

func NewServer() *Server {
	return &Server{
		Node: maelstrom.NewNode(),
	}
}

func main() {

	s := NewServer()
	s.Node.Handle("generate", s.HandleGenerate)

	if err := s.Node.Run(); err != nil {
		log.Fatal(err)
	}
}
