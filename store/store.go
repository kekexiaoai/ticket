package store

import (
	"context"
	"errors"
	"log"

	"github.com/kekexiaoai/ticket/model"
)

// TicketStore 定义存储接口
type TicketStore interface {
	SaveTicket(ctx context.Context, ticket *model.Ticket) error
	GetTicket(ctx context.Context, id string) (*model.Ticket, error)
}

// MockStore 模拟存储
type MockStore struct {
	tickets map[string]*model.Ticket
}

func NewMockStore() *MockStore {
	return &MockStore{tickets: make(map[string]*model.Ticket)}
}

func (s *MockStore) SaveTicket(ctx context.Context, ticket *model.Ticket) error {
	s.tickets[ticket.ID] = ticket
	log.Printf("保存工单: %s, 当前状态: %s, 优先级: %d", ticket.ID, ticket.CurrentState, ticket.Priority)
	return nil
}

func (s *MockStore) GetTicket(ctx context.Context, id string) (*model.Ticket, error) {
	if ticket, ok := s.tickets[id]; ok {
		return ticket, nil
	}
	return nil, errors.New("ticket not found")
}
