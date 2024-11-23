package main

import (
	"encoding/json"
	"log"
	"sync"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

type Server struct {
	Node *maelstrom.Node

	Messages   []int
	MessagesMu sync.RWMutex

	NeighbouringNodes []string
	NeighbourMutex    sync.Mutex
}

func NewServer() *Server {
	return &Server{
		Node:              maelstrom.NewNode(),
		Messages:          make([]int, 0),
		NeighbouringNodes: make([]string, 0),
	}
}

type BroadcastInput struct {
	Type    string `json:"type"`
	Message int    `json:"message"`
}

type BroadcastOutput struct {
	Type string `json:"type"`
}

func (s *Server) HandleBroadcast(msg maelstrom.Message) error {
	var body BroadcastInput
	if err := json.Unmarshal(msg.Body, &body); err != nil {
		return err
	}

	message := body.Message

	s.MessagesMu.Lock()
	s.Messages = append(s.Messages, message)
	s.MessagesMu.Unlock()

	return s.Node.Reply(msg, BroadcastOutput{
		Type: "broadcast_ok",
	})
}

func (s *Server) HandleRead(msg maelstrom.Message) error {
	var body map[string]any
	if err := json.Unmarshal(msg.Body, &body); err != nil {
		return err
	}

	body["type"] = "read_ok"

	s.MessagesMu.RLock()
	body["messages"] = s.Messages
	s.MessagesMu.RUnlock()

	return s.Node.Reply(msg, body)
}

type TopologyInput struct {
	Type     string              `json:"type"`
	Topology map[string][]string `json:"topology"`
}

type TopologyOutput struct {
	Type string `json:"type"`
}

func (s *Server) HandleTopology(msg maelstrom.Message) error {
	var body TopologyInput
	if err := json.Unmarshal(msg.Body, &body); err != nil {
		return err
	}

	s.NeighbourMutex.Lock()
	s.NeighbouringNodes = body.Topology[s.Node.ID()]
	s.NeighbourMutex.Unlock()

	topologyOutput := TopologyOutput{
		Type: "topology_ok",
	}
	return s.Node.Reply(msg, topologyOutput)
}

func main() {
	s := NewServer()

	s.Node.Handle("broadcast", s.HandleBroadcast)
	s.Node.Handle("read", s.HandleRead)
	s.Node.Handle("topology", s.HandleTopology)

	if err := s.Node.Run(); err != nil {
		log.Fatal(err)
	}
}
