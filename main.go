package main

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"

	"github.com/kekexiaoai/ticket/model"
	"github.com/kekexiaoai/ticket/service"
	"github.com/kekexiaoai/ticket/store"
	"github.com/kekexiaoai/ticket/workflow"
)

func main() {
	store := store.NewMockStore()
	ts := service.NewTicketService(store)

	// 创建工单
	ticket := &model.Ticket{
		ID:              uuid.New().String(),
		Title:           "服务器故障",
		Description:     "服务器无法启动",
		Priority:        1,
		InitialPriority: 1, // 设置初始优先级
		CurrentState:    string(workflow.StateNew),
		CreatorID:       "user123",
		CreatedAt:       time.Now(),
	}
	if err := store.SaveTicket(context.Background(), ticket); err != nil {
		log.Fatal(err)
	}

	// 执行状态转换
	ctx := context.Background()
	if err := ts.TransitionTicket(ctx, ticket.ID, workflow.EventSubmit, "user123"); err != nil {
		log.Fatal(err)
	}
	if err := ts.TransitionTicket(ctx, ticket.ID, workflow.EventAssign, "user456"); err != nil {
		log.Fatal(err)
	}
	if err := ts.TransitionTicket(ctx, ticket.ID, workflow.EventApproveInitial, "user456"); err != nil {
		log.Fatal(err)
	}
	if err := ts.TransitionTicket(ctx, ticket.ID, workflow.EventReassign, "user789"); err != nil {
		log.Fatal(err)
	}
	if err := ts.TransitionTicket(ctx, ticket.ID, workflow.EventReassign, "user999"); err != nil {
		log.Fatal(err)
	}
	if err := ts.TransitionTicket(ctx, ticket.ID, workflow.EventSubmitFinal, "user999"); err != nil {
		log.Fatal(err)
	}
	if err := ts.TransitionTicket(ctx, ticket.ID, workflow.EventApproveFinal, "admin"); err != nil {
		log.Fatal(err)
	}

	// 打印历史记录
	ticket, err := store.GetTicket(ctx, ticket.ID)
	if err != nil {
		log.Fatal(err)
	}
	ticket.PrintHistory()
}
