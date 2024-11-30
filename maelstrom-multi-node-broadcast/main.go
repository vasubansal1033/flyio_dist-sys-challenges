package main

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

const (
	SYNC_TIMEOUT = 200 * time.Millisecond
)

type Server struct {
	Node *maelstrom.Node

	Messages   map[int]bool
	MessagesMu sync.RWMutex

	NeighbouringNodes        []string
	NeighbourMutex           sync.RWMutex
	NeighbouringNodeMessages map[string]map[int]bool // each node knows what messages are present in nbr node via map[int]bool
}

func NewServer() *Server {
	return &Server{
		Node:                     maelstrom.NewNode(),
		Messages:                 make(map[int]bool),
		NeighbouringNodes:        make([]string, 0),
		NeighbouringNodeMessages: make(map[string]map[int]bool),
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
	s.Messages[message] = true
	s.MessagesMu.Unlock()

	return s.Node.Reply(msg, BroadcastOutput{
		Type: "broadcast_ok",
	})
}

type ReadInput struct {
	Type string `json:"type"`
}

type ReadOutput struct {
	Type     string `json:"type"`
	Messages []int  `json:"messages"`
}

func (s *Server) HandleRead(msg maelstrom.Message) error {
	var inputBody ReadInput
	if err := json.Unmarshal(msg.Body, &inputBody); err != nil {
		return err
	}

	s.MessagesMu.RLock()
	messages := []int{}
	for message := range s.Messages {
		messages = append(messages, message)
	}
	s.MessagesMu.RUnlock()

	return s.Node.Reply(msg, ReadOutput{
		Type:     "read_ok",
		Messages: messages,
	})
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
	for _, neighbour := range s.NeighbouringNodes {
		if _, ok := s.NeighbouringNodeMessages[neighbour]; !ok {
			s.NeighbouringNodeMessages[neighbour] = make(map[int]bool)
		}
	}
	s.NeighbourMutex.Unlock()

	topologyOutput := TopologyOutput{
		Type: "topology_ok",
	}
	return s.Node.Reply(msg, topologyOutput)
}

type SyncInput struct {
	Type     string `json:"type"`
	Messages []int  `json:"messages"`
}

type SyncOutput struct {
	Type string `json:"type"`
}

func (s *Server) HandleSync(msg maelstrom.Message) error {
	var inputBody SyncInput
	if err := json.Unmarshal(msg.Body, &inputBody); err != nil {
		return err
	}

	s.MessagesMu.Lock()
	for _, message := range inputBody.Messages {
		s.Messages[message] = true
	}
	s.MessagesMu.Unlock()

	return s.Node.Reply(msg, SyncOutput{
		Type: "sync_ok",
	})
}

func main() {
	s := NewServer()

	s.Node.Handle("broadcast", s.HandleBroadcast)
	s.Node.Handle("read", s.HandleRead)
	s.Node.Handle("topology", s.HandleTopology)
	s.Node.Handle("sync", s.HandleSync)

	done := make(chan struct{})
	// Every SYNC_TIMEOUT milliseconds, we call the sync rpc on all neighbors.
	// If there are no errors, we update the set of known messages
	// for that neighbor

	go func() {
		t := time.NewTicker(SYNC_TIMEOUT)
		for {
			select {
			case <-t.C:
				s.NeighbourMutex.RLock()
				for _, neighbour := range s.NeighbouringNodes {
					newMessages := []int{}

					s.MessagesMu.RLock()
					for message := range s.Messages {
						if !s.NeighbouringNodeMessages[neighbour][message] {
							newMessages = append(newMessages, message)
						}
					}
					s.MessagesMu.RUnlock()

					syncBody := SyncInput{
						Type:     "sync",
						Messages: newMessages,
					}

					s.Node.RPC(neighbour, syncBody, func(msg maelstrom.Message) error {
						s.NeighbourMutex.Lock()
						defer s.NeighbourMutex.Unlock()

						if msg.RPCError() != nil {
							return msg.RPCError()
						}

						for _, newMessage := range newMessages {
							s.NeighbouringNodeMessages[neighbour][newMessage] = true
						}
						return nil
					})
				}
				s.NeighbourMutex.RUnlock()
			case <-done:
				return
			}
		}
	}()

	if err := s.Node.Run(); err != nil {
		log.Fatal(err)
	}
}
