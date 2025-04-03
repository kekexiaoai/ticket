package service

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/kekexiaoai/ticket/model"
	"github.com/kekexiaoai/ticket/store"
	"github.com/kekexiaoai/ticket/workflow"
)

// TicketService 处理工单逻辑
type TicketService struct {
	sm    *workflow.StateMachine
	store store.TicketStore
}

func NewTicketService(store store.TicketStore) *TicketService {
	ts := &TicketService{
		sm:    workflow.NewStateMachine(),
		store: store,
	}
	ts.registerTasks()
	return ts
}

// 定义任务工厂
var (
	// Pending 任务
	notifyAssign = workflow.Task{
		Name: "NotifyAssign",
		Execute: func(ctx context.Context, ticket *model.Ticket, event workflow.Event) error {
			if event == workflow.EventAssign {
				log.Printf("通知: 工单 %s 被审批人 %s 领取", ticket.ID, ticket.AssigneeID)
			}
			return nil
		},
	}
	onExitPending = workflow.Task{
		Name: "OnExitPending",
		Execute: func(ctx context.Context, ticket *model.Ticket, event workflow.Event) error {
			log.Printf("退出 Pending: 工单 %s 被领取或取消", ticket.ID)
			return nil
		},
	}

	// InitialReview 任务
	notifyInitialReview = workflow.Task{
		Name: "NotifyInitialReview",
		Execute: func(ctx context.Context, ticket *model.Ticket, event workflow.Event) error {
			if event == workflow.EventApproveInitial {
				log.Printf("通知: 工单 %s 初审通过，进入处理流程", ticket.ID)
			} else if event == workflow.EventRejectInitial {
				log.Printf("通知: 工单 %s 初审被打回，需补充材料", ticket.ID)
			}
			return nil
		},
	}

	// InProgress 任务
	onEnterInProgress = workflow.Task{
		Name: "OnEnterInProgress",
		Execute: func(ctx context.Context, ticket *model.Ticket, event workflow.Event) error {
			log.Printf("进入 InProgress: 工单 %s 开始处理", ticket.ID)
			return nil
		},
	}
	checkInProgress = workflow.Task{
		Name: "CheckInProgress",
		Execute: func(ctx context.Context, ticket *model.Ticket, event workflow.Event) error {
			log.Printf("检查: 工单 %s 在 InProgress 执行 %s", ticket.ID, event)
			return nil
		},
	}
	logReassign = workflow.Task{
		Name: "LogReassign",
		Execute: func(ctx context.Context, ticket *model.Ticket, event workflow.Event) error {
			if event == workflow.EventReassign {
				log.Printf("日志: 工单 %s 被转交给 %s", ticket.ID, ticket.AssigneeID)
			}
			return nil
		},
	}
	// InProgress 任务
	updatePriority = workflow.Task{
		Name: "UpdatePriority",
		Execute: func(ctx context.Context, ticket *model.Ticket, event workflow.Event) error {
			if event == workflow.EventReassign || event == workflow.EventResume {
				newPriority := ticket.InitialPriority + ticket.ReassignCount
				if newPriority != ticket.Priority {
					ticket.Priority = newPriority
					log.Printf("任务: 工单 %s 优先级更新为 %d (转交次数: %d)", ticket.ID, ticket.Priority, ticket.ReassignCount)
				}
			}
			return nil
		},
	}

	// FinalApproval 任务
	guardFinalApproval = workflow.Task{
		Name: "GuardFinalApproval",
		Execute: func(ctx context.Context, ticket *model.Ticket, event workflow.Event) error {
			if event == workflow.EventApproveFinal && ticket.AssigneeID != "admin" {
				return errors.New("只有管理员可以最终审批")
			}
			return nil
		},
	}
	notifyFinalApproval = workflow.Task{
		Name: "NotifyFinalApproval",
		Execute: func(ctx context.Context, ticket *model.Ticket, event workflow.Event) error {
			if event == workflow.EventApproveFinal {
				log.Printf("通知: 工单 %s 最终审批通过", ticket.ID)
			}
			return nil
		},
	}
)

func (ts *TicketService) registerTasks() {
	ts.sm.RegisterTasks(workflow.StatePending, nil, []workflow.Task{notifyAssign}, nil, []workflow.Task{onExitPending}, nil)
	ts.sm.RegisterTasks(workflow.StateInitialReview, nil, []workflow.Task{notifyInitialReview}, nil, nil, nil)
	ts.sm.RegisterTasks(workflow.StateInProgress, []workflow.Task{checkInProgress}, []workflow.Task{logReassign, updatePriority}, []workflow.Task{onEnterInProgress}, nil, nil)
	ts.sm.RegisterTasks(workflow.StateFinalApproval, nil, []workflow.Task{notifyFinalApproval}, nil, nil, []workflow.Task{guardFinalApproval})
}

func (ts *TicketService) TransitionTicket(ctx context.Context, ticketID string, event workflow.Event, triggeredBy string) error {
	ticket, err := ts.store.GetTicket(ctx, ticketID)
	if err != nil {
		return err
	}
	ticket.AssigneeID = triggeredBy

	_, err = ts.sm.Transition(ctx, ticket, event)
	if err != nil {
		return err
	}

	ticket.UpdatedAt = time.Now()
	return ts.store.SaveTicket(ctx, ticket)
}
