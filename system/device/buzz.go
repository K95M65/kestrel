package device

import (
	"context"
	"fmt"
	"strings"

	"go.autonomous.ai/os/system/buzz"
	"go.autonomous.ai/os/system/server/config"
)

func (s *Service) buzzCfg() config.Buzz {
	if s == nil || s.config == nil || s.config.Buzz == nil {
		return config.Buzz{}
	}
	return *s.config.Buzz
}

// BuzzStatus is secret-free.
type BuzzStatus struct {
	Enabled bool   `json:"enabled"`
	Host    bool   `json:"host"`
	Relay   string `json:"relay_url,omitempty"`
	JoinURL string `json:"join_url,omitempty"`
	Peers   int    `json:"peers"`
	Ready   bool   `json:"ready"`
}

func (s *Service) BuzzStatus() BuzzStatus {
	b := s.buzzCfg()
	st := BuzzStatus{Enabled: b.Enabled, Host: b.Host, Relay: b.RelayURL}
	if s.networkService != nil {
		if ip, err := s.networkService.GetCurrentIP(); err == nil && ip != "" {
			st.JoinURL = "ws://" + ip + "/api/buzz/ws"
		}
	}
	s.buzzMu.Lock()
	if s.buzzHub != nil {
		st.Peers = s.buzzHub.Count()
		st.Ready = true
	}
	if s.buzzClient != nil {
		st.Ready = s.buzzClient.Connected()
	}
	s.buzzMu.Unlock()
	return st
}

func (s *Service) SetBuzz(in config.Buzz) error {
	return s.config.WithLockSave(func(c *config.Config) {
		cp := in
		c.Buzz = &cp
	})
}

func (s *Service) stopBuzz() {
	s.buzzMu.Lock()
	defer s.buzzMu.Unlock()
	if s.buzzStop != nil {
		close(s.buzzStop)
		s.buzzStop = nil
	}
	s.buzzHub = nil
	s.buzzClient = nil
}

// RestartBuzz applies the current buzz config without restarting os-server.
func (s *Service) RestartBuzz() {
	s.buzzCtl.Lock()
	defer s.buzzCtl.Unlock()
	s.stopBuzz()
	s.ensureBuzz()
}

func (s *Service) BuzzHub() *buzz.Hub {
	s.buzzMu.Lock()
	defer s.buzzMu.Unlock()
	return s.buzzHub
}

func (s *Service) SayBuzz(text string) error {
	text = strings.TrimSpace(text)
	if text == "" {
		return fmt.Errorf("empty")
	}
	id := strings.TrimSpace(s.config.DeviceID)
	name := id
	if name == "" {
		name = "kestrel"
	}
	n := buzz.Note{Type: "note", From: id, Name: name, Device: id, Text: text}
	s.buzzMu.Lock()
	hub := s.buzzHub
	cli := s.buzzClient
	s.buzzMu.Unlock()
	if hub != nil {
		hub.Broadcast(n)
		return nil
	}
	if cli != nil {
		return cli.Say(text)
	}
	return fmt.Errorf("buzz is off")
}

func (s *Service) StartBuzz(ctx context.Context) {
	s.RestartBuzz()
	<-ctx.Done()
	s.buzzCtl.Lock()
	s.stopBuzz()
	s.buzzCtl.Unlock()
}

func (s *Service) ensureBuzz() {
	b := s.buzzCfg()
	if !b.Enabled {
		return
	}
	id := strings.TrimSpace(s.config.DeviceID)
	name := id
	if name == "" {
		name = "kestrel"
	}
	onNote := func(n buzz.Note) {
		if n.From == id || strings.TrimSpace(n.Text) == "" {
			return
		}
		if s.agentGateway == nil || !s.agentGateway.IsReady() {
			return
		}
		who := n.Name
		if who == "" {
			who = n.From
		}
		msg := "[user] [buzz] " + who + " says: " + n.Text
		if _, err := s.agentGateway.SendChatMessage(msg); err != nil {
			return
		}
	}
	s.buzzMu.Lock()
	s.buzzStop = make(chan struct{})
	stop := s.buzzStop
	s.buzzMu.Unlock()
	if b.Host {
		h := buzz.NewHub()
		h.SetHandler(onNote)
		s.buzzMu.Lock()
		s.buzzHub = h
		s.buzzMu.Unlock()
	}
	url := strings.TrimSpace(b.RelayURL)
	if url != "" && !b.Host {
		cli := &buzz.Client{URL: url, Device: id, Name: name, OnNote: onNote}
		s.buzzMu.Lock()
		s.buzzClient = cli
		s.buzzMu.Unlock()
		go cli.Start(stop)
	}
}
