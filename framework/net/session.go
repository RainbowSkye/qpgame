package net

import "sync"

type Session struct {
	sync.RWMutex
	Cid  string
	Uid  string
	data map[string]any
}

func NewSession(cid string) *Session {
	return &Session{
		Cid:  cid,
		data: make(map[string]any),
	}
}

func (s *Session) Set(key string, value any) {
	s.Lock()
	defer s.Unlock()
	s.data[key] = value
}

func (s *Session) Get(key string) (any, bool) {
	s.RLocker()
	defer s.RUnlock()
	val, ok := s.data[key]
	return val, ok
}

func (s *Session) SetData(uid string, data map[string]any) {
	s.Lock()
	defer s.Unlock()
	if s.Uid == uid {
		for k, v := range data {
			s.data[k] = v
		}
	}
}
