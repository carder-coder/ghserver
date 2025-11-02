package define

import (
	"time"
)

// 任务状态常量
const (
	TaskStatusNotAccepted = 0 // 未接受
	TaskStatusAccepted    = 1 // 已接受
	TaskStatusInProgress  = 2 // 进行中
	TaskStatusCompleted   = 3 // 已完成
	TaskStatusRewarded    = 4 // 已领奖
)

// 任务类型常量
const (
	TaskTypeMain     = 1 // 主线任务
	TaskTypeDaily    = 2 // 日常任务
	TaskTypeWeekly   = 3 // 周常任务
	TaskTypeActivity = 4 // 活动任务
)

// 房间状态常量
const (
	RoomStatusWaiting = 0 // 等待中
	RoomStatusPlaying = 1 // 战斗中
	RoomStatusEnded   = 2 // 已结束
)

// 房间类型常量
const (
	RoomTypeNormal = 0 // 普通房间
	RoomTypeElite  = 1 // 精英房间
	RoomTypeBoss   = 2 // Boss房间
)

// 战斗结果常量
const (
	BattleResultLose = 0 // 失败
	BattleResultWin  = 1 // 胜利
)

// Player 玩家数据
type Player struct {
	ID            string    `bson:"_id" json:"id"`
	Username      string    `bson:"username" json:"username"`
	Password      string    `bson:"password" json:"-"` // 不输出到JSON
	Nickname      string    `bson:"nickname" json:"nickname"`
	Level         int       `bson:"level" json:"level"`
	Exp           int64     `bson:"exp" json:"exp"`
	Coin          int64     `bson:"coin" json:"coin"`
	Diamond       int64     `bson:"diamond" json:"diamond"`
	CreateTime    time.Time `bson:"create_time" json:"create_time"`
	LastLoginTime time.Time `bson:"last_login_time" json:"last_login_time"`
	OnlineStatus  bool      `bson:"online_status" json:"online_status"`
	CurrentRoomID string    `bson:"current_room_id" json:"current_room_id"`
}

// Character 角色数据
type Character struct {
	ID            string    `bson:"_id" json:"id"`
	PlayerID      string    `bson:"player_id" json:"player_id"`
	Name          string    `bson:"name" json:"name"`
	CharacterType int       `bson:"character_type" json:"character_type"`
	Level         int       `bson:"level" json:"level"`
	Power         int       `bson:"power" json:"power"`
	Health        int       `bson:"health" json:"health"`
	Attack        int       `bson:"attack" json:"attack"`
	Defense       int       `bson:"defense" json:"defense"`
	Speed         int       `bson:"speed" json:"speed"`
	CreateTime    time.Time `bson:"create_time" json:"create_time"`
}

// Item 物品数据
type Item struct {
	ID         string    `bson:"_id" json:"id"`
	PlayerID   string    `bson:"player_id" json:"player_id"`
	ItemType   int       `bson:"item_type" json:"item_type"`
	ItemID     int       `bson:"item_id" json:"item_id"`
	Count      int       `bson:"count" json:"count"`
	IsEquipped bool      `bson:"is_equipped" json:"is_equipped"`
	CreateTime time.Time `bson:"create_time" json:"create_time"`
}

// Mail 邮件数据
type Mail struct {
	ID         string     `bson:"_id" json:"id"`
	PlayerID   string     `bson:"player_id" json:"player_id"`
	Title      string     `bson:"title" json:"title"`
	Content    string     `bson:"content" json:"content"`
	Items      []ItemInfo `bson:"items" json:"items"`
	Coin       int64      `bson:"coin" json:"coin"`
	Diamond    int64      `bson:"diamond" json:"diamond"`
	IsRead     bool       `bson:"is_read" json:"is_read"`
	IsClaimed  bool       `bson:"is_claimed" json:"is_claimed"`
	SendTime   time.Time  `bson:"send_time" json:"send_time"`
	ExpireTime time.Time  `bson:"expire_time" json:"expire_time"`
}

// ItemInfo 邮件附件物品信息
type ItemInfo struct {
	ItemID int `bson:"item_id" json:"item_id"`
	Count  int `bson:"count" json:"count"`
}

// TaskDefinition 任务定义
type TaskDefinition struct {
	TaskID          string     `bson:"task_id" json:"task_id"`
	TaskName        string     `bson:"task_name" json:"task_name"`
	Description     string     `bson:"description" json:"description"`
	TaskType        int        `bson:"task_type" json:"task_type"`
	Target          int        `bson:"target" json:"target"` // 目标进度
	Rewards         []ItemInfo `bson:"rewards" json:"rewards"`
	DifficultyLevel int        `bson:"difficulty_level" json:"difficulty_level"`
	Precondition    string     `bson:"precondition" json:"precondition"` // 前置任务ID
	CreateTime      time.Time  `bson:"create_time" json:"create_time"`
	UpdateTime      time.Time  `bson:"update_time" json:"update_time"`
}

// Task 任务数据
type Task struct {
	ID           string    `bson:"_id" json:"id"`
	PlayerID     string    `bson:"player_id" json:"player_id"`
	TaskType     int       `bson:"task_type" json:"task_type"`
	TaskID       string    `bson:"task_id" json:"task_id"`
	Progress     int       `bson:"progress" json:"progress"`
	Target       int       `bson:"target" json:"target"`
	Status       int       `bson:"status" json:"status"`
	CreateTime   time.Time `bson:"create_time" json:"create_time"`
	AcceptTime   time.Time `bson:"accept_time" json:"accept_time"`
	CompleteTime time.Time `bson:"complete_time" json:"complete_time"`
	ExpireTime   time.Time `bson:"expire_time" json:"expire_time"`
}

// BattleRecord 战斗记录
type BattleRecord struct {
	ID          string    `bson:"_id" json:"id"`
	PlayerID    string    `bson:"player_id" json:"player_id"`
	RoomID      string    `bson:"room_id" json:"room_id"`
	StageID     int       `bson:"stage_id" json:"stage_id"`
	CharacterID string    `bson:"character_id" json:"character_id"`
	Result      bool      `bson:"result" json:"result"` // true: 胜利, false: 失败
	Damage      int64     `bson:"damage" json:"damage"`
	KillCount   int       `bson:"kill_count" json:"kill_count"`
	Duration    int       `bson:"duration" json:"duration"` // 战斗时长(秒)
	StartTime   time.Time `bson:"start_time" json:"start_time"`
	EndTime     time.Time `bson:"end_time" json:"end_time"`
}

// Room 房间数据
type Room struct {
	ID          string       `bson:"_id" json:"id"`
	RoomType    int          `bson:"room_type" json:"room_type"`
	StageID     int          `bson:"stage_id" json:"stage_id"`
	Status      int          `bson:"status" json:"status"` // 0: 等待中, 1: 战斗中, 2: 已结束
	PlayerCount int          `bson:"player_count" json:"player_count"`
	MaxPlayers  int          `bson:"max_players" json:"max_players"`
	Players     []PlayerInfo `bson:"players" json:"players"`
	CreateTime  time.Time    `bson:"create_time" json:"create_time"`
	StartTime   time.Time    `bson:"start_time" json:"start_time"`
	EndTime     time.Time    `bson:"end_time" json:"end_time"`
	NodeID      string       `bson:"node_id" json:"node_id"` // 所属节点服务器ID
}

// PlayerInfo 房间内玩家信息
type PlayerInfo struct {
	PlayerID    string `bson:"player_id" json:"player_id"`
	CharacterID string `bson:"character_id" json:"character_id"`
	Ready       bool   `bson:"ready" json:"ready"`
	Score       int    `bson:"score" json:"score"`
}

// Ranking 排行榜数据
type Ranking struct {
	ID         string    `bson:"_id" json:"id"`
	PlayerID   string    `bson:"player_id" json:"player_id"`
	Nickname   string    `bson:"nickname" json:"nickname"`
	RankType   int       `bson:"rank_type" json:"rank_type"` // 0: 等级, 1: 战斗力, 2: 胜率
	Score      int64     `bson:"score" json:"score"`
	UpdateTime time.Time `bson:"update_time" json:"update_time"`
}

// Position 位置信息
type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// Rotation 旋转信息
type Rotation struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// PlayerState 玩家状态
type PlayerState struct {
	PlayerID    string   `json:"player_id"`
	CharacterID int32    `json:"character_id"`
	HP          int32    `json:"hp"`
	MP          int32    `json:"mp"`
	Position    Position `json:"position"`
	Rotation    Rotation `json:"rotation"`
	WeaponID    int64    `json:"weapon_id"`
	Action      int32    `json:"action"` // 0: 站立, 1: 移动, 2: 跳跃, 3: 射击, 4: 技能, 5: 死亡
}
