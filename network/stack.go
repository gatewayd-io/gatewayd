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

	return s.items[len(s.items)-1]
}

func (s *Stack) PopLastRequest() *Request {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.items) == 0 {
		return nil
	}

	lastItem := len(s.items) - 1
	req := s.items[lastItem]
	s.items = append(s.items[:lastItem], s.items[lastItem+1:]...)
	return req
}

func (s *Stack) UpdateLastRequest(req *Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.items) == 0 {
		return
	}

	s.items[len(s.items)-1] = req
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
