package model

import (
	"fmt"
	"strings"
	"time"
)

type Ticket struct {
	ID              string    `json:"id"`
	Title           string    `json:"title"`
	Description     string    `json:"description"`
	Priority        int       `json:"priority"`
	InitialPriority int       `json:"initial_priority"` // 新增字段
	ReassignCount   int       `json:"reassign_count"`
	CurrentState    string    `json:"current_state"`
	CreatorID       string    `json:"creator_id"`
	AssigneeID      string    `json:"assignee_id"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	History         []History `json:"history"`
}

type History struct {
	FromState   string    `json:"from_state"`
	ToState     string    `json:"to_state"`
	Event       string    `json:"event"`
	Timestamp   time.Time `json:"timestamp"`
	TriggeredBy string    `json:"triggered_by"`
}

func (t *Ticket) PrintHistory() {
	if len(t.History) == 0 {
		fmt.Printf("工单 %s 无历史记录\n", t.ID)
		return
	}
	fmt.Printf("工单 %s 的历史记录 (初始优先级: %d, 当前优先级: %d, 转交次数: %d):\n", t.ID, t.InitialPriority, t.Priority, t.ReassignCount)
	fmt.Println("时间                  | 事件            | 从状态            | 到状态            | 触发者")
	fmt.Println(strings.Repeat("-", 80))
	for _, h := range t.History {
		fmt.Printf("%s | %-15s | %-17s | %-17s | %s\n",
			h.Timestamp.Format("2006-01-02 15:04:05"),
			h.Event,
			h.FromState,
			h.ToState,
			h.TriggeredBy,
		)
	}
	fmt.Println(strings.Repeat("-", 80))
}
