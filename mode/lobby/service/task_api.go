package server

import (
	"context"
	"errors"

	"ghserver/proto/pb"

	"github.com/dobyte/due/v2/cluster/node"
	"github.com/dobyte/due/v2/codes"
	"github.com/dobyte/due/v2/log"
)

// TaskServer 任务服务
type TaskServer struct {
	pb.UnimplementedTaskServiceServer
	proxy       *node.Proxy
	taskManager *TaskManager
}

func NewTaskServer(proxy *node.Proxy) *TaskServer {
	return &TaskServer{
		proxy:       proxy,
		taskManager: NewTaskManager(),
	}
}

func (s *TaskServer) Init() {
	s.proxy.AddServiceProvider("task", &pb.TaskService_ServiceDesc, s)
}

func (s *TaskServer) Close() error {
	// 清理资源
	return nil
}

func (s *TaskServer) GetTaskList(ctx context.Context, req *pb.GetTaskListRequest) (*pb.GetTaskListResponse, error) {
	log.Debugf("Get task list request: player_id=%s, task_type=%d", req.PlayerId, req.TaskType)

	// 获取任务列表
	tasks := s.taskManager.GetTaskList(req.PlayerId, int(req.TaskType))

	// 转换为响应格式
	taskBriefs := make([]*pb.TaskBrief, len(tasks))
	for i, task := range tasks {
		taskBriefs[i] = &pb.TaskBrief{
			TaskId:   int32(task.ID),
			Type:     int32(task.Type),
			Name:     task.Name,
			Status:   int32(task.Status),
			Progress: int32(task.Progress),
			Target:   int32(task.Target),
		}
	}

	return &pb.GetTaskListResponse{
		Code:    int32(codes.OK.Code()),
		Message: "获取任务列表成功",
		Tasks:   taskBriefs,
	}, nil
}

func (s *TaskServer) GetTaskDetail(ctx context.Context, req *pb.GetTaskDetailRequest) (*pb.GetTaskDetailResponse, error) {
	log.Debugf("Get task detail request: player_id=%s, task_id=%d", req.PlayerId, req.TaskId)

	// 获取任务详情
	task, err := s.taskManager.GetTaskDetail(req.PlayerId, int(req.TaskId))
	if err != nil {
		return &pb.GetTaskDetailResponse{
				Code:    int32(codes.InvalidArgument.Code()),
				Message: err.Error(),
			},
			nil
	}

	// 转换目标和进度
	targets := make(map[string]int32)
	progress := make(map[string]int32)
	for k, v := range task.Targets {
		targets[k] = int32(v)
	}
	for k, v := range task.ProgressMap {
		progress[k] = int32(v)
	}

	// 转换奖励
	rewards := make([]*pb.TaskReward, len(task.Rewards))
	for i, reward := range task.Rewards {
		rewards[i] = &pb.TaskReward{
			Type:   int32(reward.Type),
			ItemId: int32(reward.ItemID),
			Count:  int32(reward.Count),
		}
	}

	return &pb.GetTaskDetailResponse{
		Code:    int32(codes.OK.Code()),
		Message: "获取任务详情成功",
		Task: &pb.TaskDetail{
			Id:          int32(task.ID),
			Type:        int32(task.Type),
			Name:        task.Name,
			Description: task.Description,
			Status:      int32(task.Status),
			Targets:     targets,
			Progress:    progress,
			Rewards:     rewards,
			AcceptLevel: int32(task.AcceptLevel),
			NextTaskId:  int32(task.NextTaskID),
		},
	}, nil
}

func (s *TaskServer) AcceptTask(ctx context.Context, req *pb.AcceptTaskRequest) (*pb.CommonResponse, error) {
	log.Debugf("Accept task request: player_id=%s, task_id=%d", req.PlayerId, req.TaskId)

	// 接受任务
	err := s.taskManager.AcceptTask(req.PlayerId, int(req.TaskId))
	if err != nil {
		return &pb.CommonResponse{
				Code:    int32(codes.InvalidArgument.Code()),
				Message: err.Error(),
			},
			nil
	}

	return &pb.CommonResponse{
		Code:    int32(codes.OK.Code()),
		Message: "接受任务成功",
	}, nil
}

func (s *TaskServer) SubmitTask(ctx context.Context, req *pb.SubmitTaskRequest) (*pb.SubmitTaskResponse, error) {
	log.Debugf("Submit task request: player_id=%s, task_id=%d", req.PlayerId, req.TaskId)

	// 提交任务
	rewards, err := s.taskManager.SubmitTask(req.PlayerId, int(req.TaskId))
	if err != nil {
		return &pb.SubmitTaskResponse{
				Code:    int32(codes.InvalidArgument.Code()),
				Message: err.Error(),
			},
			nil
	}

	// 转换奖励格式
	taskRewards := make([]*pb.TaskReward, len(rewards))
	for i, reward := range rewards {
		taskRewards[i] = &pb.TaskReward{
			Type:   int32(reward.Type),
			ItemId: int32(reward.ItemID),
			Count:  int32(reward.Count),
		}
	}

	return &pb.SubmitTaskResponse{
		Code:    int32(codes.OK.Code()),
		Message: "提交任务成功",
		Rewards: taskRewards,
	}, nil
}

func (s *TaskServer) GiveUpTask(ctx context.Context, req *pb.GiveUpTaskRequest) (*pb.CommonResponse, error) {
	log.Debugf("Give up task request: player_id=%s, task_id=%d", req.PlayerId, req.TaskId)

	// 放弃任务
	err := s.taskManager.GiveUpTask(req.PlayerId, int(req.TaskId))
	if err != nil {
		return &pb.CommonResponse{
				Code:    int32(codes.InvalidArgument.Code()),
				Message: err.Error(),
			},
			nil
	}

	return &pb.CommonResponse{
		Code:    int32(codes.OK.Code()),
		Message: "放弃任务成功",
	}, nil
}

// Task 任务结构
type Task struct {
	ID          int
	Type        int
	Name        string
	Description string
	Status      int // 1未接取，2进行中，3已完成，4已提交
	Targets     map[string]int
	ProgressMap map[string]int
	Progress    int // 进度百分比
	Target      int // 目标值
	Rewards     []TaskReward
	AcceptLevel int
	NextTaskID  int
}

// TaskReward 任务奖励
type TaskReward struct {
	Type   int
	ItemID int
	Count  int
}

// TaskManager 任务管理器
type TaskManager struct {
	playerTasks map[string][]Task // player_id -> tasks
	globalTasks map[int]*Task     // task_id -> template
}

func NewTaskManager() *TaskManager {
	m := &TaskManager{
		playerTasks: make(map[string][]Task),
		globalTasks: make(map[int]*Task),
	}

	// 初始化全局任务模板
	m.initGlobalTasks()

	return m
}

func (m *TaskManager) GetTaskList(playerID string, taskType int) []*Task {
	// 初始化玩家任务
	if _, exists := m.playerTasks[playerID]; !exists {
		m.initPlayerTasks(playerID)
	}

	tasks := m.playerTasks[playerID]
	result := make([]*Task, 0)

	for i := range tasks {
		// 根据任务类型筛选
		if taskType == 0 || tasks[i].Type == taskType {
			result = append(result, &tasks[i])
		}
	}

	return result
}

func (m *TaskManager) GetTaskDetail(playerID string, taskID int) (*Task, error) {
	// 初始化玩家任务
	if _, exists := m.playerTasks[playerID]; !exists {
		m.initPlayerTasks(playerID)
	}

	for i := range m.playerTasks[playerID] {
		if m.playerTasks[playerID][i].ID == taskID {
			return &m.playerTasks[playerID][i], nil
		}
	}

	return nil, errors.New("任务不存在")
}

func (m *TaskManager) AcceptTask(playerID string, taskID int) error {
	// 初始化玩家任务
	if _, exists := m.playerTasks[playerID]; !exists {
		m.initPlayerTasks(playerID)
	}

	for i := range m.playerTasks[playerID] {
		if m.playerTasks[playerID][i].ID == taskID {
			if m.playerTasks[playerID][i].Status != 1 {
				return errors.New("任务已接取或已完成")
			}

			// 检查等级限制（这里假设玩家等级为1）
			playerLevel := 1
			if playerLevel < m.playerTasks[playerID][i].AcceptLevel {
				return errors.New("等级不足")
			}

			// 标记为进行中
			m.playerTasks[playerID][i].Status = 2
			return nil
		}
	}

	return errors.New("任务不存在")
}

func (m *TaskManager) SubmitTask(playerID string, taskID int) ([]TaskReward, error) {
	// 初始化玩家任务
	if _, exists := m.playerTasks[playerID]; !exists {
		m.initPlayerTasks(playerID)
	}

	for i := range m.playerTasks[playerID] {
		if m.playerTasks[playerID][i].ID == taskID {
			if m.playerTasks[playerID][i].Status != 3 {
				return nil, errors.New("任务未完成，无法提交")
			}

			// 标记为已提交
			rewards := m.playerTasks[playerID][i].Rewards
			m.playerTasks[playerID][i].Status = 4

			// 如果有后续任务，解锁后续任务
			if m.playerTasks[playerID][i].NextTaskID > 0 {
				for j := range m.playerTasks[playerID] {
					if m.playerTasks[playerID][j].ID == m.playerTasks[playerID][i].NextTaskID {
						m.playerTasks[playerID][j].Status = 1
						break
					}
				}
			}

			return rewards, nil
		}
	}

	return nil, errors.New("任务不存在")
}

func (m *TaskManager) GiveUpTask(playerID string, taskID int) error {
	// 初始化玩家任务
	if _, exists := m.playerTasks[playerID]; !exists {
		m.initPlayerTasks(playerID)
	}

	for i := range m.playerTasks[playerID] {
		if m.playerTasks[playerID][i].ID == taskID {
			if m.playerTasks[playerID][i].Status != 2 {
				return errors.New("只能放弃进行中的任务")
			}

			// 重置为未接取
			m.playerTasks[playerID][i].Status = 1
			// 重置进度
			m.playerTasks[playerID][i].Progress = 0
			for k := range m.playerTasks[playerID][i].ProgressMap {
				m.playerTasks[playerID][i].ProgressMap[k] = 0
			}

			return nil
		}
	}

	return errors.New("任务不存在")
}

func (m *TaskManager) initGlobalTasks() {
	// 主线任务
	m.globalTasks[1001] = &Task{
		ID:          1001,
		Type:        1,
		Name:        "初入江湖",
		Description: "与村长对话，了解村庄的情况",
		Status:      1,
		Targets:     map[string]int{"talk_npc": 1},
		ProgressMap: map[string]int{"talk_npc": 0},
		Progress:    0,
		Target:      1,
		Rewards: []TaskReward{
			{Type: 1, ItemID: 0, Count: 100},  // 经验
			{Type: 2, ItemID: 0, Count: 1000}, // 金币
		},
		AcceptLevel: 1,
		NextTaskID:  1002,
	}

	m.globalTasks[1002] = &Task{
		ID:          1002,
		Type:        1,
		Name:        "除掉山贼",
		Description: "消灭10个山贼，保护村庄安全",
		Status:      0, // 初始锁定
		Targets:     map[string]int{"kill_monster": 10},
		ProgressMap: map[string]int{"kill_monster": 0},
		Progress:    0,
		Target:      10,
		Rewards: []TaskReward{
			{Type: 1, ItemID: 0, Count: 200},
			{Type: 2, ItemID: 0, Count: 2000},
			{Type: 3, ItemID: 2001, Count: 1}, // 装备
		},
		AcceptLevel: 2,
		NextTaskID:  0,
	}

	// 支线任务
	m.globalTasks[2001] = &Task{
		ID:          2001,
		Type:        2,
		Name:        "收集药材",
		Description: "帮郎中收集5株草药",
		Status:      1,
		Targets:     map[string]int{"collect_herb": 5},
		ProgressMap: map[string]int{"collect_herb": 0},
		Progress:    0,
		Target:      5,
		Rewards: []TaskReward{
			{Type: 1, ItemID: 0, Count: 50},
			{Type: 2, ItemID: 0, Count: 500},
		},
		AcceptLevel: 1,
		NextTaskID:  0,
	}

	// 日常任务
	m.globalTasks[3001] = &Task{
		ID:          3001,
		Type:        3,
		Name:        "日常巡逻",
		Description: "在村庄周围巡逻1次",
		Status:      1,
		Targets:     map[string]int{"patrol": 1},
		ProgressMap: map[string]int{"patrol": 0},
		Progress:    0,
		Target:      1,
		Rewards: []TaskReward{
			{Type: 1, ItemID: 0, Count: 30},
			{Type: 2, ItemID: 0, Count: 300},
		},
		AcceptLevel: 1,
		NextTaskID:  0,
	}
}

func (m *TaskManager) initPlayerTasks(playerID string) {
	// 复制全局任务模板到玩家任务
	playerTasks := make([]Task, 0)
	for _, taskTemplate := range m.globalTasks {
		task := *taskTemplate
		// 深拷贝map
		task.Targets = make(map[string]int)
		for k, v := range taskTemplate.Targets {
			task.Targets[k] = v
		}
		task.ProgressMap = make(map[string]int)
		for k, v := range taskTemplate.ProgressMap {
			task.ProgressMap[k] = v
		}
		// 深拷贝rewards
		task.Rewards = make([]TaskReward, len(taskTemplate.Rewards))
		copy(task.Rewards, taskTemplate.Rewards)

		playerTasks = append(playerTasks, task)
	}

	// 为示例方便，将一些任务标记为已完成
	for i := range playerTasks {
		if playerTasks[i].ID == 1001 {
			playerTasks[i].Progress = 100
			playerTasks[i].ProgressMap["talk_npc"] = 1
			playerTasks[i].Status = 3
		}
		if playerTasks[i].ID == 2001 {
			playerTasks[i].Progress = 60
			playerTasks[i].ProgressMap["collect_herb"] = 3
			playerTasks[i].Status = 2
		}
	}

	m.playerTasks[playerID] = playerTasks
}
