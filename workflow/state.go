package workflow

import (
	"context"
	"errors"
	"time"

	"github.com/kekexiaoai/ticket/model"
)

// State 定义工单状态
type State string

const (
	StateNew           State = "New"
	StatePending       State = "Pending"
	StateInitialReview State = "InitialReview"
	StateInProgress    State = "InProgress"
	StateFinalApproval State = "FinalApproval"
	StateCompleted     State = "Completed"
	StateClosed        State = "Closed"
	StateCanceled      State = "Canceled"
)

// Event 定义触发状态转换的事件
type Event string

const (
	EventSubmit         Event = "Submit"
	EventAssign         Event = "Assign"
	EventApproveInitial Event = "ApproveInitial"
	EventRejectInitial  Event = "RejectInitial"
	EventDenyInitial    Event = "DenyInitial"
	EventSubmitFinal    Event = "SubmitFinal"
	EventApproveFinal   Event = "ApproveFinal"
	EventRejectFinal    Event = "RejectFinal"
	EventArchive        Event = "Archive"
	EventCancel         Event = "Cancel"
	EventReassign       Event = "Reassign"
	EventHold           Event = "Hold"
	EventResume         Event = "Resume"
)

// Task 定义任务
type Task struct {
	Name    string
	Execute func(ctx context.Context, ticket *model.Ticket, event Event) error
}

// Node 定义工作流节点
type Node struct {
	State       State
	BeforeTasks []Task
	AfterTasks  []Task
	OnEnter     []Task // 进入状态时
	OnExit      []Task // 退出状态时
	Guards      []Task // 转换条件检查
}

// StateMachine 状态机
type StateMachine struct {
	transitions map[State]map[Event]State
	nodes       map[State]*Node
}

func NewStateMachine() *StateMachine {
	sm := &StateMachine{
		transitions: make(map[State]map[Event]State),
		nodes:       make(map[State]*Node),
	}
	sm.initTransitions()
	sm.initNodes()
	return sm
}

func (sm *StateMachine) initTransitions() {
	sm.transitions[StateNew] = map[Event]State{EventSubmit: StatePending}
	sm.transitions[StatePending] = map[Event]State{
		EventAssign: StateInitialReview,
		EventCancel: StateCanceled,
	}
	sm.transitions[StateInitialReview] = map[Event]State{
		EventApproveInitial: StateInProgress,
		EventRejectInitial:  StateNew,
		EventDenyInitial:    StateCanceled,
	}
	sm.transitions[StateInProgress] = map[Event]State{
		EventSubmitFinal: StateFinalApproval,
		EventReassign:    StateInProgress,
		EventHold:        StateInProgress,
		EventResume:      StateInProgress,
	}
	sm.transitions[StateFinalApproval] = map[Event]State{
		EventApproveFinal: StateCompleted,
		EventRejectFinal:  StateInProgress,
	}
	sm.transitions[StateCompleted] = map[Event]State{EventArchive: StateClosed}
}

func (sm *StateMachine) initNodes() {
	sm.nodes[StateNew] = &Node{State: StateNew}
	sm.nodes[StatePending] = &Node{State: StatePending}
	sm.nodes[StateInitialReview] = &Node{State: StateInitialReview}
	sm.nodes[StateInProgress] = &Node{State: StateInProgress}
	sm.nodes[StateFinalApproval] = &Node{State: StateFinalApproval}
	sm.nodes[StateCompleted] = &Node{State: StateCompleted}
	sm.nodes[StateClosed] = &Node{State: StateClosed}
	sm.nodes[StateCanceled] = &Node{State: StateCanceled}
}

// RegisterTasks 注册任务
func (sm *StateMachine) RegisterTasks(state State, before, after, onEnter, onExit, guards []Task) {
	node, ok := sm.nodes[state]
	if !ok {
		node = &Node{State: state}
		sm.nodes[state] = node
	}
	node.BeforeTasks = append(node.BeforeTasks, before...)
	node.AfterTasks = append(node.AfterTasks, after...)
	node.OnEnter = append(node.OnEnter, onEnter...)
	node.OnExit = append(node.OnExit, onExit...)
	node.Guards = append(node.Guards, guards...)
}

func (sm *StateMachine) Transition(ctx context.Context, ticket *model.Ticket, event Event) (State, error) {
	currentState := State(ticket.CurrentState)
	nextState, ok := sm.transitions[currentState][event]
	if !ok {
		return currentState, errors.New("invalid transition")
	}

	// 执行 Guard 检查
	if node, ok := sm.nodes[currentState]; ok {
		for _, guard := range node.Guards {
			if err := guard.Execute(ctx, ticket, event); err != nil {
				return currentState, err
			}
		}
	}

	// 执行 Before 任务
	if node, ok := sm.nodes[currentState]; ok {
		for _, task := range node.BeforeTasks {
			if err := task.Execute(ctx, ticket, event); err != nil {
				return currentState, err
			}
		}
	}

	// 执行 OnExit 任务
	if node, ok := sm.nodes[currentState]; ok {
		for _, task := range node.OnExit {
			if err := task.Execute(ctx, ticket, event); err != nil {
				return currentState, err
			}
		}
	}

	// 更新状态和 ReassignCount
	oldState := ticket.CurrentState
	ticket.CurrentState = string(nextState)
	if event == EventReassign {
		ticket.ReassignCount++
	}
	ticket.UpdatedAt = time.Now()
	ticket.History = append(ticket.History, model.History{
		FromState:   string(oldState),
		ToState:     string(nextState),
		Event:       string(event),
		Timestamp:   time.Now(),
		TriggeredBy: ticket.AssigneeID,
	})

	// 执行 OnEnter 任务
	if node, ok := sm.nodes[nextState]; ok {
		for _, task := range node.OnEnter {
			if err := task.Execute(ctx, ticket, event); err != nil {
				return nextState, err
			}
		}
	}

	// 执行 After 任务
	if node, ok := sm.nodes[nextState]; ok {
		for _, task := range node.AfterTasks {
			if err := task.Execute(ctx, ticket, event); err != nil {
				return nextState, err
			}
		}
	}

	return nextState, nil
}
