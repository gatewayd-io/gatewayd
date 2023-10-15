package network

import "sync"

type Request struct {
	Data []byte
}

type Stack struct {
	items []*Request
	mu    sync.RWMutex
}

func (s *Stack) Push(req *Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.items = append(s.items, req)
}

func (s *Stack) GetLastRequest() *Request {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.items) == 0 {
		return nil
	}

	//nolint:staticcheck
	for i := len(s.items) - 1; i >= 0; i-- {
		return s.items[i]
	}

	return nil
}

func (s *Stack) PopLastRequest() *Request {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.items) == 0 {
		return nil
	}

	//nolint:staticcheck
	for i := len(s.items) - 1; i >= 0; i-- {
		req := s.items[i]
		s.items = append(s.items[:i], s.items[i+1:]...)
		return req
	}

	return nil
}

func (s *Stack) UpdateLastRequest(req *Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.items) == 0 {
		return
	}

	//nolint:staticcheck
	for i := len(s.items) - 1; i >= 0; i-- {
		s.items[i] = req
		return
	}
}

func (s *Stack) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.items = make([]*Request, 0)
}

func NewStack() *Stack {
	return &Stack{
		items: make([]*Request, 0),
		mu:    sync.RWMutex{},
	}
}
