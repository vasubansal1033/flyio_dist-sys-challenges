package main

import (
	"context"
	"encoding/json"
	"log"
	"sync"

	maelstorm "github.com/jepsen-io/maelstrom/demo/go"
)

type Server struct {
	Node *maelstorm.Node
	KV   *maelstorm.KV

	counter   int
	counterMu sync.Mutex
}

type AddInput struct {
	Type  string `json:"type"`
	Delta int    `json:"delta"`
}

type AddOutput struct {
	Type string `json:"type"`
}

func (s *Server) addHandler(msg maelstorm.Message) (err error) {
	var inputBody AddInput
	if err := json.Unmarshal(msg.Body, &inputBody); err != nil {
		return err
	}

	s.counterMu.Lock()

	ctx := context.Background()
	err = s.KV.CompareAndSwap(ctx, s.Node.ID(), s.counter, s.counter+inputBody.Delta, true)
	if err != nil {
		return err
	}
	s.counter += inputBody.Delta
	s.counterMu.Unlock()

	return s.Node.Reply(msg, AddOutput{
		Type: "add_ok",
	})
}

type ReadInput struct {
	Type string `json:"type"`
}

type ReadOutput struct {
	Type  string `json:"type"`
	Value int    `json:"value"`
}

func (s *Server) readHandler(msg maelstorm.Message) (err error) {
	var inputBody ReadInput
	if err := json.Unmarshal(msg.Body, &inputBody); err != nil {
		return err
	}

	totalNodes := len(s.Node.NodeIDs())
	total := make(chan int, totalNodes)
	total <- s.counter

	for _, id := range s.Node.NodeIDs() {
		if id != s.Node.ID() {
			go func() {
				val, err := s.KV.ReadInt(context.Background(), id)
				if err != nil {
					val = 0
				}

				total <- val
			}()
		}
	}

	result := 0
	for i := 0; i < totalNodes; i++ {
		result += <-total
	}

	return s.Node.Reply(msg, ReadOutput{
		Type:  "read_ok",
		Value: result,
	})
}

func NewServer() *Server {
	node := maelstorm.NewNode()
	return &Server{
		Node: node,
		KV:   maelstorm.NewSeqKV(node),
	}
}

func main() {

	s := NewServer()
	s.Node.Handle("read", s.readHandler)
	s.Node.Handle("add", s.addHandler)

	if err := s.Node.Run(); err != nil {
		log.Fatal(err)
	}
}
