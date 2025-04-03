package service

import (
	"context"
	"testing"
	"time"

	"github.com/kekexiaoai/ticket/model"
	"github.com/kekexiaoai/ticket/store"
	"github.com/kekexiaoai/ticket/workflow"
)

func TestTicketService_TransitionTicket(t *testing.T) {
	store := store.NewMockStore()
	ts := NewTicketService(store)

	// 初始化工单
	ticket := &model.Ticket{
		ID:           "test-ticket",
		Title:        "Test Ticket",
		Priority:     1,
		CurrentState: string(workflow.StateNew),
		CreatorID:    "user123",
		CreatedAt:    time.Now(),
	}
	if err := store.SaveTicket(context.Background(), ticket); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name        string
		event       workflow.Event
		triggeredBy string
		wantState   string
		wantErr     bool
	}{
		{"Submit", workflow.EventSubmit, "user123", string(workflow.StatePending), false},
		{"Assign", workflow.EventAssign, "user456", string(workflow.StateInitialReview), false},
		{"ApproveInitial", workflow.EventApproveInitial, "user456", string(workflow.StateInProgress), false},
		{"Reassign", workflow.EventReassign, "user789", string(workflow.StateInProgress), false},
		{"Invalid Event", workflow.EventCancel, "user789", string(workflow.StateInProgress), true},
		{"FinalApproval with non-admin", workflow.EventSubmitFinal, "user789", string(workflow.StateFinalApproval), false}, // 进入 FinalApproval
		{"FinalApproval fail", workflow.EventApproveFinal, "user789", string(workflow.StateFinalApproval), true},           // Guard 阻止
		{"FinalApproval success", workflow.EventApproveFinal, "admin", string(workflow.StateCompleted), false},             // Guard 通过
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ts.TransitionTicket(context.Background(), ticket.ID, tt.event, tt.triggeredBy)
			if (err != nil) != tt.wantErr {
				t.Errorf("TransitionTicket() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				updatedTicket, _ := store.GetTicket(context.Background(), ticket.ID)
				if updatedTicket.CurrentState != tt.wantState {
					t.Errorf("Ticket.CurrentState = %v, want %v", updatedTicket.CurrentState, tt.wantState)
				}
			}
		})
	}
}

func TestTicketService_InProgressPriorityTask(t *testing.T) {
	store := store.NewMockStore()
	ts := NewTicketService(store)

	ticket := &model.Ticket{
		ID:              "test-ticket",
		Title:           "Test Ticket",
		Priority:        1,
		InitialPriority: 1, // 设置初始优先级
		CurrentState:    string(workflow.StateNew),
		CreatorID:       "user123",
		CreatedAt:       time.Now(),
	}
	if err := store.SaveTicket(context.Background(), ticket); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	ts.TransitionTicket(ctx, ticket.ID, workflow.EventSubmit, "user123")
	ts.TransitionTicket(ctx, ticket.ID, workflow.EventAssign, "user456")
	ts.TransitionTicket(ctx, ticket.ID, workflow.EventApproveInitial, "user456")

	// 第一次 Reassign
	ts.TransitionTicket(ctx, ticket.ID, workflow.EventReassign, "user789")
	updatedTicket, _ := store.GetTicket(ctx, ticket.ID)
	if updatedTicket.ReassignCount != 1 {
		t.Errorf("ReassignCount = %d, want 1", updatedTicket.ReassignCount)
	}
	if updatedTicket.Priority != 2 {
		t.Errorf("Priority = %d, want 2 after first Reassign", updatedTicket.Priority)
	}

	// 第二次 Reassign
	ts.TransitionTicket(ctx, ticket.ID, workflow.EventReassign, "user999")
	updatedTicket, _ = store.GetTicket(ctx, ticket.ID)
	if updatedTicket.ReassignCount != 2 {
		t.Errorf("ReassignCount = %d, want 2", updatedTicket.ReassignCount)
	}
	if updatedTicket.Priority != 3 {
		t.Errorf("Priority = %d, want 3 after second Reassign", updatedTicket.Priority)
	}
}

func TestTicketService_PendingOnExit(t *testing.T) {
	store := store.NewMockStore()
	ts := NewTicketService(store)

	ticket := &model.Ticket{
		ID:           "test-ticket",
		Title:        "Test Ticket",
		Priority:     1,
		CurrentState: string(workflow.StateNew),
		CreatorID:    "user123",
		CreatedAt:    time.Now(),
	}
	if err := store.SaveTicket(context.Background(), ticket); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	ts.TransitionTicket(ctx, ticket.ID, workflow.EventSubmit, "user123")
	ts.TransitionTicket(ctx, ticket.ID, workflow.EventAssign, "user456")

	updatedTicket, _ := store.GetTicket(ctx, ticket.ID)
	if updatedTicket.CurrentState != string(workflow.StateInitialReview) {
		t.Errorf("Ticket.CurrentState = %v, want %v", updatedTicket.CurrentState, workflow.StateInitialReview)
	}
	// OnExitPending 的效果通过日志验证（这里假设日志已记录）
}
