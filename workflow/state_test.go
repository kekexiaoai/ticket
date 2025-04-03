package workflow

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kekexiaoai/ticket/model"
)

func TestStateMachine_Transition(t *testing.T) {
	tests := []struct {
		name      string
		current   State
		event     Event
		wantState State
		wantErr   bool
	}{
		{"New to Pending", StateNew, EventSubmit, StatePending, false},
		{"Pending to InitialReview", StatePending, EventAssign, StateInitialReview, false},
		{"Pending to Canceled", StatePending, EventCancel, StateCanceled, false},
		{"InitialReview to InProgress", StateInitialReview, EventApproveInitial, StateInProgress, false},
		{"InitialReview to New", StateInitialReview, EventRejectInitial, StateNew, false},
		{"InitialReview to Canceled", StateInitialReview, EventDenyInitial, StateCanceled, false},
		{"InProgress to FinalApproval", StateInProgress, EventSubmitFinal, StateFinalApproval, false},
		{"InProgress self-loop (Reassign)", StateInProgress, EventReassign, StateInProgress, false},
		{"FinalApproval to Completed", StateFinalApproval, EventApproveFinal, StateCompleted, false},
		{"FinalApproval to InProgress", StateFinalApproval, EventRejectFinal, StateInProgress, false},
		{"Completed to Closed", StateCompleted, EventArchive, StateClosed, false},
		{"Invalid transition", StateNew, EventCancel, StateNew, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := NewStateMachine()
			ticket := &model.Ticket{
				ID:           "test-ticket",
				CurrentState: string(tt.current),
				CreatedAt:    time.Now(),
			}

			gotState, err := sm.Transition(context.Background(), ticket, tt.event)
			if (err != nil) != tt.wantErr {
				t.Errorf("Transition() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotState != tt.wantState {
				t.Errorf("Transition() gotState = %v, want %v", gotState, tt.wantState)
			}
			if ticket.CurrentState != string(tt.wantState) {
				t.Errorf("Ticket.CurrentState = %v, want %v", ticket.CurrentState, tt.wantState)
			}
		})
	}
}

func TestStateMachine_Hooks(t *testing.T) {
	sm := NewStateMachine()
	ticket := &model.Ticket{
		ID:           "test-ticket",
		CurrentState: string(StatePending),
		AssigneeID:   "user123",
		CreatedAt:    time.Now(),
	}

	// 注册钩子
	var guardCalled, beforeCalled, onExitCalled, onEnterCalled, afterCalled bool
	sm.RegisterTasks(StatePending,
		[]Task{{Name: "Before", Execute: func(ctx context.Context, ticket *model.Ticket, event Event) error {
			beforeCalled = true
			return nil
		}}},
		nil,
		nil,
		[]Task{{Name: "OnExit", Execute: func(ctx context.Context, ticket *model.Ticket, event Event) error {
			onExitCalled = true
			return nil
		}}},
		[]Task{{Name: "Guard", Execute: func(ctx context.Context, ticket *model.Ticket, event Event) error {
			guardCalled = true
			return nil
		}}},
	)
	sm.RegisterTasks(StateInitialReview,
		nil,
		[]Task{{Name: "After", Execute: func(ctx context.Context, ticket *model.Ticket, event Event) error {
			afterCalled = true
			return nil
		}}},
		[]Task{{Name: "OnEnter", Execute: func(ctx context.Context, ticket *model.Ticket, event Event) error {
			onEnterCalled = true
			return nil
		}}},
		nil,
		nil,
	)

	// 执行转换
	_, err := sm.Transition(context.Background(), ticket, EventAssign)
	if err != nil {
		t.Fatalf("Transition() error = %v", err)
	}

	// 验证钩子执行
	if !guardCalled {
		t.Error("Guard task was not called")
	}
	if !beforeCalled {
		t.Error("Before task was not called")
	}
	if !onExitCalled {
		t.Error("OnExit task was not called")
	}
	if !onEnterCalled {
		t.Error("OnEnter task was not called")
	}
	if !afterCalled {
		t.Error("After task was not called")
	}
	if ticket.CurrentState != string(StateInitialReview) {
		t.Errorf("Ticket.CurrentState = %v, want %v", ticket.CurrentState, StateInitialReview)
	}
}

func TestStateMachine_GuardFailure(t *testing.T) {
	sm := NewStateMachine()
	ticket := &model.Ticket{
		ID:           "test-ticket",
		CurrentState: string(StateFinalApproval),
		AssigneeID:   "user123", // 非管理员
		CreatedAt:    time.Now(),
	}

	// 注册 Guard 任务（模拟权限检查）
	sm.RegisterTasks(StateFinalApproval,
		nil,
		nil,
		nil,
		nil,
		[]Task{{Name: "GuardFinalApproval", Execute: func(ctx context.Context, ticket *model.Ticket, event Event) error {
			if event == EventApproveFinal && ticket.AssigneeID != "admin" {
				return errors.New("只有管理员可以最终审批")
			}
			return nil
		}}},
	)

	_, err := sm.Transition(context.Background(), ticket, EventApproveFinal)
	if err == nil {
		t.Error("Expected error from Guard, got nil")
	}
	if ticket.CurrentState != string(StateFinalApproval) {
		t.Errorf("Ticket.CurrentState = %v, want %v", ticket.CurrentState, StateFinalApproval)
	}
}

func TestStateMachine_OnEnterFailure(t *testing.T) {
	sm := NewStateMachine()
	ticket := &model.Ticket{
		ID:           "test-ticket",
		CurrentState: string(StatePending),
		AssigneeID:   "user123",
		CreatedAt:    time.Now(),
	}

	// 注册 OnEnter 失败任务
	sm.RegisterTasks(StateInitialReview,
		nil,
		nil,
		[]Task{{Name: "OnEnterFail", Execute: func(ctx context.Context, ticket *model.Ticket, event Event) error {
			return errors.New("OnEnter failed")
		}}},
		nil,
		nil,
	)

	_, err := sm.Transition(context.Background(), ticket, EventAssign)
	if err == nil {
		t.Error("Expected error from OnEnter, got nil")
	}
	if ticket.CurrentState != string(StateInitialReview) {
		t.Errorf("Ticket.CurrentState = %v, want %v", ticket.CurrentState, StateInitialReview)
	}
}
