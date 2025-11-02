package server

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"time"

	pb "ghserver/proto/pb"
	"ghserver/utils/mongodb"

	"github.com/dobyte/due/v2/cluster/mesh"
	"github.com/dobyte/due/v2/codes"
	"github.com/dobyte/due/v2/log"
	"github.com/dobyte/due/v2/utils/xconv"
	"github.com/dobyte/due/v2/utils/xtime"
	"golang.org/x/crypto/bcrypt"
)

type LoginServer struct {
	pb.UnimplementedLoginServiceServer
	proxy         *mesh.Proxy
	mongoClient   *mongodb.MongoDBClient
	playerManager *PlayerManager
	queueManager  *QueueManager
}

func NewLoginServer(proxy *mesh.Proxy) *LoginServer {
	return &LoginServer{
		proxy:         proxy,
		playerManager: NewPlayerManager(),
		queueManager:  NewQueueManager(),
	}
}
func (s *LoginServer) Close() error {
	return nil
}
func (s *LoginServer) Init() {
	s.proxy.AddServiceProvider("login", &pb.LoginService_ServiceDesc, s)
	// 创建MongoDB客户端
	s.mongoClient, _ = mongodb.NewMongoDBClient("game", "user")

	// 启动队列处理协程
	s.queueManager.Start()
}

// Register 注册
func (s *LoginServer) Register(ctx context.Context, args *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	user, err := s.doQueryUserByAccount(ctx, args.Account)
	if err != nil {
		return nil, err
	}

	if user != nil {
		return nil, errors.NewError(code.AccountExists)
	}

	password, err := bcrypt.GenerateFromPassword([]byte(args.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Errorf("generate password failed:%v", err)
		return nil, errors.NewError(err, code.InternalError)
	}

	_, err = s.userDao.Insert(ctx, &model.User{
		Account:     args.Account,
		Password:    xconv.String(password),
		Nickname:    args.Nickname,
		RegisterAt:  xtime.Now(),
		RegisterIP:  args.ClientIP,
		LastLoginAt: xtime.Now(),
		LastLoginIP: args.ClientIP,
	})
	if err != nil {
		log.Errorf("insert user failed: %v", err)
		return nil, errors.NewError(err, code.InternalError)
	}

	return &pb.RegisterReply{}, nil
}

func (s *LoginServer) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	log.Debugf("Login request: account=%s, device_id=%s", req.Account, req.DeviceId)

	// 验证账号密码（这里简化处理）
	if req.Account == "" || req.Password == "" {
		return &pb.LoginResponse{
			Code:    int32(codes.InvalidArgument.Code()),
			Message: "账号或密码不能为空",
		}, nil
	}

	// 检查是否已经在线
	if s.playerManager.IsOnline(req.Account) {
		// 可以选择踢出旧连接或者拒绝新连接
		player := s.playerManager.GetPlayerByAccount(req.Account)
		if player != nil {
			log.Infof("Player %s is already online, kicking old connection", req.Account)
			// 这里可以调用踢人逻辑
		}
	}

	// 模拟查询玩家信息
	playerInfo := &pb.PlayerInfo{
		Id:       fmt.Sprintf("player_%s", req.Account),
		Nickname: fmt.Sprintf("Player_%s", req.Account),
		Level:    1,
		Exp:      0,
		VipLevel: 0,
		Items:    []int32{1001, 1002},
	}

	// 检查是否需要排队
	if s.playerManager.GetOnlineCount() >= 100 {
		// 需要排队
		position := s.queueManager.AddToQueue(req.Account, playerInfo)
		log.Infof("Player %s added to queue, position: %d", req.Account, position)
		return &pb.LoginResponse{
			Code:          int32(codes.OK.Code()),
			Message:       "您已进入排队队列",
			QueuePosition: int32(position),
		}, nil
	}

	// 生成token
	token := s.generateToken(req.Account)

	// 创建玩家对象并标记为在线
	player := &Player{
		Account:   req.Account,
		PlayerID:  playerInfo.Id,
		Token:     token,
		DeviceID:  req.DeviceId,
		LoginTime: time.Now(),
	}
	s.playerManager.AddPlayer(player)

	log.Infof("Player %s logged in successfully", req.Account)

	return &pb.LoginResponse{
		Code:    int32(codes.OK.Code()),
		Message: "登录成功",
		Token:   token,
		Player:  playerInfo,
	}, nil
}

func (s *LoginServer) Reconnect(ctx context.Context, req *pb.ReconnectRequest) (*pb.LoginResponse, error) {
	log.Debugf("Reconnect request: token=%s, device_id=%s", req.Token, req.DeviceId)

	// 验证token
	player := s.playerManager.GetPlayerByToken(req.Token)
	if player == nil {
		return &pb.LoginResponse{
			Code:    int32(codes.Unauthorized.Code()),
			Message: "token无效，请重新登录",
		}, nil
	}

	// 验证设备ID
	if player.DeviceID != req.DeviceId {
		log.Warnf("Device ID mismatch for player %s", player.Account)
		return &pb.LoginResponse{
			Code:    int32(codes.Unknown.Code()),
			Message: "设备不匹配，请使用原设备登录",
		}, nil
	}

	// 模拟查询玩家信息
	playerInfo := &pb.PlayerInfo{
		Id:       player.PlayerID,
		Nickname: fmt.Sprintf("Player_%s", player.Account),
		Level:    1,
		Exp:      0,
		VipLevel: 0,
		Items:    []int32{1001, 1002},
	}

	// 更新重连时间
	player.ReconnectTime = time.Now()

	log.Infof("Player %s reconnected successfully", player.Account)

	return &pb.LoginResponse{
		Code:    int32(codes.OK.Code()),
		Message: "重连成功",
		Token:   player.Token,
		Player:  playerInfo,
	}, nil
}

func (s *LoginServer) KickPlayer(ctx context.Context, req *pb.KickPlayerRequest) (*pb.CommonResponse, error) {
	log.Debugf("Kick player request: player_id=%s, reason=%s", req.PlayerId, req.Reason)

	// 查找玩家
	player := s.playerManager.GetPlayerByID(req.PlayerId)
	if player == nil {
		return &pb.CommonResponse{
			Code:    int32(codes.NotFound.Code()),
			Message: "玩家不存在",
		}, nil
	}

	// 踢出玩家
	s.playerManager.RemovePlayer(player.Account)

	// 这里可以发送踢出消息给玩家
	log.Infof("Player %s kicked out, reason: %s", req.PlayerId, req.Reason)

	return &pb.CommonResponse{
		Code:    int32(codes.OK.Code()),
		Message: "踢出成功",
	}, nil
}

func (s *LoginServer) GetQueueInfo(ctx context.Context, req *pb.QueueInfoRequest) (*pb.QueueInfoResponse, error) {
	log.Debugf("Get queue info request: token=%s", req.Token)

	// 验证token获取玩家信息
	player := s.playerManager.GetPlayerByToken(req.Token)
	if player == nil {
		return &pb.QueueInfoResponse{
			Code:    int32(codes.Unauthorized.Code()),
			Message: "token无效",
		}, nil
	}

	// 获取排队信息
	position, total, err := s.queueManager.GetQueueInfo(player.Account)
	if err != nil {
		return &pb.QueueInfoResponse{
			Code:    int32(codes.NotFound.Code()),
			Message: "您不在排队队列中",
		}, nil
	}

	// 估算等待时间（假设每个玩家处理需要5秒）
	estimatedTime := position * 5

	return &pb.QueueInfoResponse{
		Code:          int32(codes.OK.Code()),
		Message:       "查询成功",
		Position:      int32(position),
		Total:         int32(total),
		EstimatedTime: int32(estimatedTime),
	}, nil
}

func (s *LoginServer) generateToken(account string) string {
	// 生成8位随机字符串
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	randomStr := make([]byte, 8)
	for i := range randomStr {
		randomStr[i] = charset[rand.Intn(len(charset))]
	}
	return fmt.Sprintf("%s_%s_%d", account, string(randomStr), time.Now().UnixNano())
}

// Player 玩家对象
type Player struct {
	Account       string
	PlayerID      string
	Token         string
	DeviceID      string
	LoginTime     time.Time
	ReconnectTime time.Time
	IsOnline      bool
}

// PlayerManager 玩家管理器
type PlayerManager struct {
	playersByAccount map[string]*Player
	playersByToken   map[string]*Player
	playersByID      map[string]*Player
	onlineCount      int
}

func NewPlayerManager() *PlayerManager {
	return &PlayerManager{
		playersByAccount: make(map[string]*Player),
		playersByToken:   make(map[string]*Player),
		playersByID:      make(map[string]*Player),
	}
}

func (pm *PlayerManager) AddPlayer(player *Player) {
	pm.playersByAccount[player.Account] = player
	pm.playersByToken[player.Token] = player
	pm.playersByID[player.PlayerID] = player
	pm.onlineCount++
}

func (pm *PlayerManager) RemovePlayer(account string) {
	player, exists := pm.playersByAccount[account]
	if !exists {
		return
	}

	delete(pm.playersByAccount, account)
	delete(pm.playersByToken, player.Token)
	delete(pm.playersByID, player.PlayerID)
	pm.onlineCount--
}

func (pm *PlayerManager) GetPlayerByAccount(account string) *Player {
	return pm.playersByAccount[account]
}

func (pm *PlayerManager) GetPlayerByToken(token string) *Player {
	return pm.playersByToken[token]
}

func (pm *PlayerManager) GetPlayerByID(playerID string) *Player {
	return pm.playersByID[playerID]
}

func (pm *PlayerManager) IsOnline(account string) bool {
	_, exists := pm.playersByAccount[account]
	return exists
}

func (pm *PlayerManager) GetOnlineCount() int {
	return pm.onlineCount
}

// QueueItem 排队项
type QueueItem struct {
	Account    string
	PlayerInfo *pb.PlayerInfo
	JoinTime   time.Time
}

// QueueManager 排队管理器
type QueueManager struct {
	queue []*QueueItem
}

func NewQueueManager() *QueueManager {
	return &QueueManager{
		queue: make([]*QueueItem, 0),
	}
}

func (qm *QueueManager) Start() {
	// 启动队列处理协程
	go func() {
		for {
			time.Sleep(500 * time.Millisecond)
			// 这里可以实现队列处理逻辑
		}
	}()
}

func (qm *QueueManager) AddToQueue(account string, playerInfo *pb.PlayerInfo) int {
	item := &QueueItem{
		Account:    account,
		PlayerInfo: playerInfo,
		JoinTime:   time.Now(),
	}
	qm.queue = append(qm.queue, item)
	return len(qm.queue)
}

func (qm *QueueManager) GetQueueInfo(account string) (position, total int, err error) {
	total = len(qm.queue)
	for i, item := range qm.queue {
		if item.Account == account {
			position = i + 1
			return
		}
	}
	err = errors.New("player not in queue")
	return
}

func (qm *QueueManager) RemoveFromQueue(account string) {
	for i, item := range qm.queue {
		if item.Account == account {
			qm.queue = append(qm.queue[:i], qm.queue[i+1:]...)
			break
		}
	}
}
