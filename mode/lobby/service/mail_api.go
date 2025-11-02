package server

import (
	"context"
	"errors"
	"time"

	"ghserver/proto/pb"

	"github.com/dobyte/due/v2/cluster/node"
	"github.com/dobyte/due/v2/codes"
	"github.com/dobyte/due/v2/log"
)

// MailServer 邮箱服务
type MailServer struct {
	pb.UnimplementedMailServiceServer
	proxy       *node.Proxy
	mailManager *MailManager
}

func NewMailServer(proxy *node.Proxy) *MailServer {
	return &MailServer{
		proxy:       proxy,
		mailManager: NewMailManager(),
	}
}

func (s *MailServer) Init() {
	s.proxy.AddServiceProvider("mail", &pb.MailService_ServiceDesc, s)
}

func (s *MailServer) Close() error {
	// 清理资源
	return nil
}

func (s *MailServer) GetMailList(ctx context.Context, req *pb.GetMailListRequest) (*pb.GetMailListResponse, error) {
	log.Debugf("Get mail list request: player_id=%s, page=%d, page_size=%d", req.PlayerId, req.Page, req.PageSize)

	// 获取邮件列表
	mails, total := s.mailManager.GetMailList(req.PlayerId, int(req.Page), int(req.PageSize))

	// 转换为响应格式
	mailBriefs := make([]*pb.MailBrief, len(mails))
	for i, mail := range mails {
		mailBriefs[i] = &pb.MailBrief{
			MailId:        mail.ID,
			Title:         mail.Title,
			SendTime:      mail.SendTime,
			HasAttachment: len(mail.Attachments) > 0,
			IsRead:        mail.IsRead,
			IsReceived:    mail.IsReceived,
		}
	}

	return &pb.GetMailListResponse{
		Code:    int32(codes.OK.Code()),
		Message: "获取邮件列表成功",
		Mails:   mailBriefs,
		Total:   int32(total),
	}, nil
}

func (s *MailServer) GetMailDetail(ctx context.Context, req *pb.GetMailDetailRequest) (*pb.GetMailDetailResponse, error) {
	log.Debugf("Get mail detail request: player_id=%s, mail_id=%s", req.PlayerId, req.MailId)

	// 获取邮件详情
	mail, err := s.mailManager.GetMailDetail(req.PlayerId, req.MailId)
	if err != nil {
		return &pb.GetMailDetailResponse{
			Code:    int32(codes.InvalidArgument.Code()),
				Message: err.Error(),
			},
			nil
	}

	// 标记为已读
	s.mailManager.MarkAsRead(req.PlayerId, req.MailId)

	// 转换附件
	attachments := make([]*pb.MailAttachment, len(mail.Attachments))
	for i, attach := range mail.Attachments {
		attachments[i] = &pb.MailAttachment{
			Type:   int32(attach.Type),
			ItemId: int32(attach.ItemID),
			Count:  int32(attach.Count),
		}
	}

	return &pb.GetMailDetailResponse{
		Code:    int32(codes.OK.Code()),
		Message: "获取邮件详情成功",
		Mail: &pb.Mail{
			Id:          mail.ID,
			Title:       mail.Title,
			Content:     mail.Content,
			SendTime:    mail.SendTime,
			ExpireTime:  mail.ExpireTime,
			IsRead:      mail.IsRead,
			IsReceived:  mail.IsReceived,
			Attachments: attachments,
		},
	}, nil
}

func (s *MailServer) ReceiveMailAttachment(ctx context.Context, req *pb.ReceiveMailAttachmentRequest) (*pb.ReceiveMailAttachmentResponse, error) {
	log.Debugf("Receive mail attachment request: player_id=%s, mail_id=%s", req.PlayerId, req.MailId)

	// 领取附件
	attachments, err := s.mailManager.ReceiveAttachment(req.PlayerId, req.MailId)
	if err != nil {
		return &pb.ReceiveMailAttachmentResponse{
			Code:    int32(codes.InvalidArgument.Code()),
				Message: err.Error(),
			},
			nil
	}

	// 转换附件格式
	mailAttachments := make([]*pb.MailAttachment, len(attachments))
	for i, attach := range attachments {
		mailAttachments[i] = &pb.MailAttachment{
			Type:   int32(attach.Type),
			ItemId: int32(attach.ItemID),
			Count:  int32(attach.Count),
		}
	}

	return &pb.ReceiveMailAttachmentResponse{
		Code:        int32(codes.OK.Code()),
		Message:     "领取附件成功",
		Attachments: mailAttachments,
	}, nil
}

func (s *MailServer) DeleteMail(ctx context.Context, req *pb.DeleteMailRequest) (*pb.CommonResponse, error) {
	log.Debugf("Delete mail request: player_id=%s, mail_ids=%v", req.PlayerId, req.MailIds)

	// 删除邮件
	err := s.mailManager.DeleteMail(req.PlayerId, req.MailIds)
	if err != nil {
		return &pb.CommonResponse{
			Code:    int32(codes.InvalidArgument.Code()),
				Message: err.Error(),
			},
			nil
	}

	return &pb.CommonResponse{
		Code:    int32(codes.OK.Code()),
		Message: "删除邮件成功",
	}, nil
}

// Mail 邮件结构
type Mail struct {
	ID          string
	Title       string
	Content     string
	SendTime    int64
	ExpireTime  int64
	IsRead      bool
	IsReceived  bool
	Attachments []MailAttachment
}

// MailAttachment 邮件附件
type MailAttachment struct {
	Type   int
	ItemID int
	Count  int
}

// MailManager 邮件管理器
type MailManager struct {
	mails map[string]map[string]*Mail // player_id -> mail_id -> Mail
}

func NewMailManager() *MailManager {
	return &MailManager{
		mails: make(map[string]map[string]*Mail),
	}
}

func (m *MailManager) GetMailList(playerID string, page, pageSize int) ([]*Mail, int) {
	playerMails, exists := m.mails[playerID]
	if !exists {
		// 为新玩家生成一些示例邮件
		m.generateExampleMails(playerID)
		playerMails = m.mails[playerID]
	}

	// 过滤过期邮件
	validMails := make([]*Mail, 0)
	now := time.Now().Unix()
	for _, mail := range playerMails {
		if mail.ExpireTime > now {
			validMails = append(validMails, mail)
		}
	}

	// 分页
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > len(validMails) {
		return []*Mail{}, len(validMails)
	}
	if end > len(validMails) {
		end = len(validMails)
	}

	return validMails[start:end], len(validMails)
}

func (m *MailManager) GetMailDetail(playerID, mailID string) (*Mail, error) {
	playerMails, exists := m.mails[playerID]
	if !exists {
		return nil, errors.New("玩家不存在")
	}

	mail, exists := playerMails[mailID]
	if !exists {
		return nil, errors.New("邮件不存在")
	}

	if mail.ExpireTime < time.Now().Unix() {
		return nil, errors.New("邮件已过期")
	}

	return mail, nil
}

func (m *MailManager) MarkAsRead(playerID, mailID string) {
	if playerMails, exists := m.mails[playerID]; exists {
		if mail, exists := playerMails[mailID]; exists {
			mail.IsRead = true
		}
	}
}

func (m *MailManager) ReceiveAttachment(playerID, mailID string) ([]MailAttachment, error) {
	mail, err := m.GetMailDetail(playerID, mailID)
	if err != nil {
		return nil, err
	}

	if mail.IsReceived {
		return nil, errors.New("附件已领取")
	}

	if len(mail.Attachments) == 0 {
		return nil, errors.New("邮件没有附件")
	}

	// 标记为已领取
	mail.IsReceived = true

	return mail.Attachments, nil
}

func (m *MailManager) DeleteMail(playerID string, mailIDs []string) error {
	playerMails, exists := m.mails[playerID]
	if !exists {
		return errors.New("玩家不存在")
	}

	for _, mailID := range mailIDs {
		delete(playerMails, mailID)
	}

	return nil
}

func (m *MailManager) generateExampleMails(playerID string) {
	now := time.Now().Unix()
	weekLater := now + 7*24*3600

	m.mails[playerID] = map[string]*Mail{
		"1": {
			ID:         "1",
			Title:      "欢迎加入游戏",
			Content:    "亲爱的玩家，欢迎加入我们的游戏！这里是你的新手礼包，请查收。",
			SendTime:   now,
			ExpireTime: weekLater,
			IsRead:     false,
			IsReceived: false,
			Attachments: []MailAttachment{
				{Type: 1, ItemID: 1001, Count: 10}, // 金币
				{Type: 2, ItemID: 2001, Count: 1},  // 装备
			},
		},
		"2": {
			ID:         "2",
			Title:      "每日签到奖励",
			Content:    "恭喜你完成每日签到，这是你的奖励。",
			SendTime:   now - 86400,
			ExpireTime: weekLater,
			IsRead:     false,
			IsReceived: false,
			Attachments: []MailAttachment{
				{Type: 1, ItemID: 1002, Count: 5}, // 钻石
			},
		},
		"3": {
			ID:          "3",
			Title:       "系统通知",
			Content:     "系统将于明天凌晨2点进行维护，请提前下线。",
			SendTime:    now - 172800,
			ExpireTime:  now + 86400,
			IsRead:      false,
			IsReceived:  true,
			Attachments: []MailAttachment{},
		},
	}
}
