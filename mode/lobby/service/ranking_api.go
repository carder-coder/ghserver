package server

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	"ghserver/proto/pb"

	"github.com/dobyte/due/v2/cluster/node"
	"github.com/dobyte/due/v2/codes"
	"github.com/dobyte/due/v2/log"
)

// RankingServer 排行榜服务
type RankingServer struct {
	pb.UnimplementedRankingServiceServer
	proxy          *node.Proxy
	rankingManager *RankingManager
}

func NewRankingServer(proxy *node.Proxy) *RankingServer {
	return &RankingServer{
		proxy:          proxy,
		rankingManager: NewRankingManager(),
	}
}

func (s *RankingServer) Init() {
	s.proxy.AddServiceProvider("ranking", &pb.RankingService_ServiceDesc, s)
}

func (s *RankingServer) Close() error {
	// 清理资源
	return nil
}

func (s *RankingServer) GetRankingList(ctx context.Context, req *pb.GetRankingListRequest) (*pb.GetRankingListResponse, error) {
	log.Debugf("Get ranking list request: ranking_type=%d, page=%d, page_size=%d", req.RankingType, req.Page, req.PageSize)

	// 获取排行榜列表
	rankings, total := s.rankingManager.GetRankingList(int32(req.RankingType), int(req.Page), int(req.PageSize))

	// 转换排行榜项
	rankingItems := make([]*pb.RankingItem, len(rankings))
	for i, rank := range rankings {
		rankingItems[i] = &pb.RankingItem{
			PlayerId: rank.PlayerID,
			Nickname: rank.Nickname,
			Rank:     int32(rank.Rank),
			Score:    int32(rank.Score),
			Level:    int32(rank.Level),
			Avatar:   rank.Avatar,
		}
	}

	return &pb.GetRankingListResponse{
		Code:    int32(codes.OK.Code()),
		Message: "获取排行榜成功",
		Items:   rankingItems,
		Total:   int32(total),
	}, nil
}

func (s *RankingServer) GetPlayerRank(ctx context.Context, req *pb.GetPlayerRankRequest) (*pb.GetPlayerRankResponse, error) {
	log.Debugf("Get player rank request: player_id=%s, ranking_type=%d", req.PlayerId, req.RankingType)

	// 获取玩家排名
	rank, err := s.rankingManager.GetPlayerRank(req.PlayerId, int32(req.RankingType))
	if err != nil {
		return &pb.GetPlayerRankResponse{
				Code:    int32(codes.InvalidArgument.Code()),
				Message: err.Error(),
			},
			nil
	}

	return &pb.GetPlayerRankResponse{
		Code:    int32(codes.OK.Code()),
		Message: "获取玩家排名成功",
		Item: &pb.RankingItem{
			PlayerId: rank.PlayerID,
			Nickname: rank.Nickname,
			Rank:     int32(rank.Rank),
			Score:    int32(rank.Score),
			Level:    int32(rank.Level),
			Avatar:   rank.Avatar,
		},
	}, nil
}

// RankingItem 排行榜项
type RankingItem struct {
	PlayerID   string
	Nickname   string
	Rank       int
	Score      int
	Level      int
	Avatar     string
	UpdateTime int64
}

// RankingList 排行榜列表
type RankingList struct {
	Type    int32
	Items   []*RankingItem
	Players map[string]*RankingItem // 快速查找玩家排名
	mutex   sync.RWMutex
}

// RankingManager 排行榜管理器
type RankingManager struct {
	rankings map[int32]*RankingList
	mutex    sync.RWMutex
}

func NewRankingManager() *RankingManager {
	manager := &RankingManager{
		rankings: make(map[int32]*RankingList),
	}

	// 初始化各类型排行榜
	manager.initRankings()

	return manager
}

func (m *RankingManager) GetRankingList(rankingType int32, page, pageSize int) ([]*RankingItem, int) {
	m.mutex.RLock()
	rankingList, exists := m.rankings[rankingType]
	m.mutex.RUnlock()

	if !exists {
		return []*RankingItem{}, 0
	}

	rankingList.mutex.RLock()
	defer rankingList.mutex.RUnlock()

	// 分页
	total := len(rankingList.Items)
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	// 获取分页数据
	result := make([]*RankingItem, 0)
	for i := start; i < end && i < total; i++ {
		result = append(result, rankingList.Items[i])
	}

	return result, total
}

func (m *RankingManager) GetPlayerRank(playerID string, rankingType int32) (*RankingItem, error) {
	m.mutex.RLock()
	rankingList, exists := m.rankings[rankingType]
	m.mutex.RUnlock()

	if !exists {
		return nil, errors.New("排行榜类型不存在")
	}

	rankingList.mutex.RLock()
	defer rankingList.mutex.RUnlock()

	item, exists := rankingList.Players[playerID]
	if !exists {
		return nil, errors.New("玩家未上榜")
	}

	return item, nil
}

// UpdatePlayerScore 更新玩家分数并重新排名
func (m *RankingManager) UpdatePlayerScore(playerID string, nickname string, avatar string, level int, score int, rankingType int32) {
	m.mutex.Lock()
	rankingList, exists := m.rankings[rankingType]
	if !exists {
		rankingList = &RankingList{
			Type:    rankingType,
			Items:   make([]*RankingItem, 0),
			Players: make(map[string]*RankingItem),
		}
		m.rankings[rankingType] = rankingList
	}
	m.mutex.Unlock()

	rankingList.mutex.Lock()
	defer rankingList.mutex.Unlock()

	// 查找玩家是否已在排行榜中
	item, exists := rankingList.Players[playerID]
	if exists {
		// 更新分数
		item.Score = score
		item.Nickname = nickname
		item.Level = level
		item.Avatar = avatar
		item.UpdateTime = time.Now().Unix()
	} else {
		// 添加新玩家
		item = &RankingItem{
			PlayerID:   playerID,
			Nickname:   nickname,
			Score:      score,
			Level:      level,
			Avatar:     avatar,
			UpdateTime: time.Now().Unix(),
		}
		rankingList.Items = append(rankingList.Items, item)
		rankingList.Players[playerID] = item
	}

	// 重新排序
	m.sortRankingList(rankingList)
}

func (m *RankingManager) sortRankingList(rankingList *RankingList) {
	// 根据分数降序排序
	sort.Slice(rankingList.Items, func(i, j int) bool {
		return rankingList.Items[i].Score > rankingList.Items[j].Score
	})

	// 更新排名
	for i := range rankingList.Items {
		rankingList.Items[i].Rank = i + 1
	}
}

func (m *RankingManager) initRankings() {
	now := time.Now().Unix()

	// 初始化等级排行榜
	levelRanking := &RankingList{
		Type:    1,
		Items:   make([]*RankingItem, 0),
		Players: make(map[string]*RankingItem),
	}

	// 添加示例数据
	levelPlayers := []*RankingItem{
		{PlayerID: "player101", Nickname: "战神", Level: 100, Score: 100, Avatar: "avatar1", UpdateTime: now},
		{PlayerID: "player102", Nickname: "法神", Level: 98, Score: 98, Avatar: "avatar2", UpdateTime: now},
		{PlayerID: "player103", Nickname: "道士", Level: 97, Score: 97, Avatar: "avatar3", UpdateTime: now},
		{PlayerID: "player104", Nickname: "猎手", Level: 95, Score: 95, Avatar: "avatar4", UpdateTime: now},
		{PlayerID: "player105", Nickname: "刺客", Level: 93, Score: 93, Avatar: "avatar5", UpdateTime: now},
		{PlayerID: "player106", Nickname: "战士", Level: 92, Score: 92, Avatar: "avatar6", UpdateTime: now},
		{PlayerID: "player107", Nickname: "法师", Level: 90, Score: 90, Avatar: "avatar7", UpdateTime: now},
		{PlayerID: "player108", Nickname: "牧师", Level: 88, Score: 88, Avatar: "avatar8", UpdateTime: now},
		{PlayerID: "player109", Nickname: "德鲁伊", Level: 85, Score: 85, Avatar: "avatar9", UpdateTime: now},
		{PlayerID: "player110", Nickname: "盗贼", Level: 82, Score: 82, Avatar: "avatar10", UpdateTime: now},
	}

	for _, player := range levelPlayers {
		levelRanking.Items = append(levelRanking.Items, player)
		levelRanking.Players[player.PlayerID] = player
	}

	m.rankings[int32(1)] = levelRanking
	m.sortRankingList(levelRanking)

	// 初始化战力排行榜
	powerRanking := &RankingList{
		Type:    int32(2),
		Items:   make([]*RankingItem, 0),
		Players: make(map[string]*RankingItem),
	}

	// 添加示例数据
	powerPlayers := []*RankingItem{
		{PlayerID: "player201", Nickname: "无敌战神", Level: 95, Score: 25000, Avatar: "avatar11", UpdateTime: now},
		{PlayerID: "player202", Nickname: "力量之王", Level: 94, Score: 24500, Avatar: "avatar12", UpdateTime: now},
		{PlayerID: "player203", Nickname: "魔法大师", Level: 93, Score: 24000, Avatar: "avatar13", UpdateTime: now},
		{PlayerID: "player204", Nickname: "敏捷猎手", Level: 92, Score: 23500, Avatar: "avatar14", UpdateTime: now},
		{PlayerID: "player205", Nickname: "暗影刺客", Level: 91, Score: 23000, Avatar: "avatar15", UpdateTime: now},
		{PlayerID: "player206", Nickname: "光明使者", Level: 90, Score: 22500, Avatar: "avatar16", UpdateTime: now},
		{PlayerID: "player207", Nickname: "自然守护者", Level: 89, Score: 22000, Avatar: "avatar17", UpdateTime: now},
		{PlayerID: "player208", Nickname: "元素使", Level: 88, Score: 21500, Avatar: "avatar18", UpdateTime: now},
		{PlayerID: "player209", Nickname: "死亡骑士", Level: 87, Score: 21000, Avatar: "avatar19", UpdateTime: now},
		{PlayerID: "player210", Nickname: "圣骑士", Level: 86, Score: 20500, Avatar: "avatar20", UpdateTime: now},
	}

	for _, player := range powerPlayers {
		powerRanking.Items = append(powerRanking.Items, player)
		powerRanking.Players[player.PlayerID] = player
	}

	m.rankings[int32(2)] = powerRanking
	m.sortRankingList(powerRanking)

	// 初始化财富排行榜
	wealthRanking := &RankingList{
		Type:    int32(3),
		Items:   make([]*RankingItem, 0),
		Players: make(map[string]*RankingItem),
	}

	// 添加示例数据
	wealthPlayers := []*RankingItem{
		{PlayerID: "player301", Nickname: "富翁", Level: 85, Score: 500000, Avatar: "avatar21", UpdateTime: now},
		{PlayerID: "player302", Nickname: "富豪", Level: 80, Score: 450000, Avatar: "avatar22", UpdateTime: now},
		{PlayerID: "player303", Nickname: "财主", Level: 75, Score: 400000, Avatar: "avatar23", UpdateTime: now},
		{PlayerID: "player304", Nickname: "商人", Level: 70, Score: 350000, Avatar: "avatar24", UpdateTime: now},
		{PlayerID: "player305", Nickname: "理财师", Level: 65, Score: 300000, Avatar: "avatar25", UpdateTime: now},
	}

	for _, player := range wealthPlayers {
		wealthRanking.Items = append(wealthRanking.Items, player)
		wealthRanking.Players[player.PlayerID] = player
	}

	m.rankings[int32(3)] = wealthRanking
	m.sortRankingList(wealthRanking)

	// 初始化击杀排行榜
	killRanking := &RankingList{
		Type:    int32(4),
		Items:   make([]*RankingItem, 0),
		Players: make(map[string]*RankingItem),
	}

	// 添加示例数据
	killPlayers := []*RankingItem{
		{PlayerID: "player401", Nickname: "杀人王", Level: 90, Score: 5000, Avatar: "avatar26", UpdateTime: now},
		{PlayerID: "player402", Nickname: "刽子手", Level: 88, Score: 4500, Avatar: "avatar27", UpdateTime: now},
		{PlayerID: "player403", Nickname: "狙击手", Level: 85, Score: 4000, Avatar: "avatar28", UpdateTime: now},
		{PlayerID: "player404", Nickname: "终结者", Level: 82, Score: 3500, Avatar: "avatar29", UpdateTime: now},
		{PlayerID: "player405", Nickname: "屠夫", Level: 80, Score: 3000, Avatar: "avatar30", UpdateTime: now},
	}

	for _, player := range killPlayers {
		killRanking.Items = append(killRanking.Items, player)
		killRanking.Players[player.PlayerID] = player
	}

	m.rankings[int32(4)] = killRanking
	m.sortRankingList(killRanking)
}
