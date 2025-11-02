package server

import (
	"context"
	"errors"
	"fmt"
	"time"

	"ghserver/proto/pb"

	"github.com/dobyte/due/v2/cluster/node"
	"github.com/dobyte/due/v2/codes"
	"github.com/dobyte/due/v2/log"
)

// ShopServer 商场服务
type ShopServer struct {
	pb.UnimplementedShopServiceServer
	proxy       *node.Proxy
	shopManager *ShopManager
}

func NewShopServer(proxy *node.Proxy) *ShopServer {
	return &ShopServer{
		proxy:       proxy,
		shopManager: NewShopManager(),
	}
}

func (s *ShopServer) Init() {
	s.proxy.AddServiceProvider("shop", &pb.ShopService_ServiceDesc, s)
}

func (s *ShopServer) Close() error {
	// 清理资源
	return nil
}

func (s *ShopServer) GetShopList(ctx context.Context, req *pb.GetShopListRequest) (*pb.GetShopListResponse, error) {
	log.Debugf("Get shop list request: player_id=%s, shop_type=%d", req.PlayerId, req.ShopType)

	// 获取商店列表
	items := s.shopManager.GetShopList(req.PlayerId, int(req.ShopType))

	// 转换为响应格式
	shopItems := make([]*pb.ShopItem, len(items))
	for i, item := range items {
		shopItems[i] = &pb.ShopItem{
			ItemId:        int32(item.ItemID),
			OriginalPrice: int32(item.OriginalPrice),
			CurrentPrice:  int32(item.CurrentPrice),
			CurrencyType:  int32(item.CurrencyType),
			Stock:         int32(item.Stock),
			MaxBuyCount:   int32(item.MaxBuyCount),
			BoughtCount:   int32(item.BoughtCount),
			ExpireTime:    item.ExpireTime,
			IsDiscount:    item.IsDiscount,
			DiscountRate:  item.DiscountRate,
		}
	}

	return &pb.GetShopListResponse{
		Code:    int32(codes.OK.Code()),
		Message: "获取商店列表成功",
		Items:   shopItems,
	}, nil
}

func (s *ShopServer) BuyItem(ctx context.Context, req *pb.BuyItemRequest) (*pb.BuyItemResponse, error) {
	log.Debugf("Buy item request: player_id=%s, item_id=%d, count=%d", req.PlayerId, req.ItemId, req.Count)

	// 购买物品
	boughtItems, spentCurrency, currencyType, err := s.shopManager.BuyItem(req.PlayerId, int(req.ItemId), int(req.Count))
	if err != nil {
		return &pb.BuyItemResponse{
				Code:    int32(codes.InvalidArgument.Code()),
				Message: err.Error(),
			},
			nil
	}

	// 转换购买的物品
	bagItems := make([]*pb.BagItem, len(boughtItems))
	for i, item := range boughtItems {
		bagItems[i] = &pb.BagItem{
			Id:         item.ID,
			ItemId:     item.ItemID,
			Count:      item.Count,
			CreateTime: item.CreateTime,
			Attrs:      item.Attrs,
		}
	}

	return &pb.BuyItemResponse{
		Code:          int32(codes.OK.Code()),
		Message:       "购买成功",
		BoughtItems:   bagItems,
		SpentCurrency: int32(spentCurrency),
		CurrencyType:  int32(currencyType),
	}, nil
}

func (s *ShopServer) GetDiscountInfo(ctx context.Context, req *pb.GetDiscountInfoRequest) (*pb.GetDiscountInfoResponse, error) {
	log.Debugf("Get discount info request: player_id=%s", req.PlayerId)

	// 获取折扣信息
	discounts := s.shopManager.GetDiscountInfo(req.PlayerId)

	// 转换为响应格式
	discountInfos := make([]*pb.DiscountInfo, len(discounts))
	for i, discount := range discounts {
		discountInfos[i] = &pb.DiscountInfo{
			ShopType:     int32(discount.ShopType),
			DiscountRate: discount.DiscountRate,
			StartTime:    discount.StartTime,
			EndTime:      discount.EndTime,
		}
	}

	return &pb.GetDiscountInfoResponse{
		Code:      int32(codes.OK.Code()),
		Message:   "获取折扣信息成功",
		Discounts: discountInfos,
	}, nil
}

// ShopItem 商店物品
type ShopItem struct {
	ItemID        int
	OriginalPrice int
	CurrentPrice  int
	CurrencyType  int // 1金币，2钻石
	Stock         int
	MaxBuyCount   int
	BoughtCount   int
	ExpireTime    int64
	IsDiscount    bool
	DiscountRate  float32
}

// DiscountInfo 折扣信息
type DiscountInfo struct {
	ShopType     int
	DiscountRate float32
	StartTime    int64
	EndTime      int64
}

// Item 背包物品
type Item struct {
	ID         string            // 物品实例ID
	ItemID     int32             // 物品ID
	Count      int32             // 数量
	CreateTime int64             // 创建时间
	Attrs      map[string]string // 物品属性
}

// PlayerCurrency 玩家货币
type PlayerCurrency struct {
	Coins int
	Gems  int
}

// ShopManager 商场管理器
type ShopManager struct {
	shopItems      map[int][]*ShopItem    // shop_type -> items
	playerBuys     map[string]map[int]int // player_id -> item_id -> bought_count
	playerCurrency map[string]*PlayerCurrency
}

func NewShopManager() *ShopManager {
	m := &ShopManager{
		shopItems:      make(map[int][]*ShopItem),
		playerBuys:     make(map[string]map[int]int),
		playerCurrency: make(map[string]*PlayerCurrency),
	}

	// 初始化商店物品
	m.initShopItems()

	return m
}

func (m *ShopManager) GetShopList(playerID string, shopType int) []*ShopItem {
	// 初始化玩家购买记录
	if _, exists := m.playerBuys[playerID]; !exists {
		m.playerBuys[playerID] = make(map[int]int)
	}

	// 初始化玩家货币
	if _, exists := m.playerCurrency[playerID]; !exists {
		m.playerCurrency[playerID] = &PlayerCurrency{
			Coins: 10000,
			Gems:  1000,
		}
	}

	items, exists := m.shopItems[shopType]
	if !exists {
		return []*ShopItem{}
	}

	// 克隆物品并更新购买数量
	result := make([]*ShopItem, len(items))
	now := time.Now().Unix()

	for i, item := range items {
		// 克隆物品
		result[i] = &ShopItem{}
		*result[i] = *item
		result[i].BoughtCount = m.playerBuys[playerID][item.ItemID]

		// 检查限时物品是否过期
		if item.ExpireTime > 0 && item.ExpireTime < now {
			result[i].Stock = 0
		}

		// 应用折扣
		if item.IsDiscount && item.ExpireTime > now {
			result[i].CurrentPrice = int(float32(item.OriginalPrice) * item.DiscountRate)
		} else {
			result[i].CurrentPrice = item.OriginalPrice
			result[i].IsDiscount = false
		}
	}

	return result
}

func (m *ShopManager) BuyItem(playerID string, itemID, count int) ([]Item, int, int, error) {
	// 初始化玩家数据
	if _, exists := m.playerBuys[playerID]; !exists {
		m.playerBuys[playerID] = make(map[int]int)
	}
	if _, exists := m.playerCurrency[playerID]; !exists {
		m.playerCurrency[playerID] = &PlayerCurrency{
			Coins: 10000,
			Gems:  1000,
		}
	}

	// 查找物品
	var targetItem *ShopItem
	for _, items := range m.shopItems {
		for _, item := range items {
			if item.ItemID == itemID {
				targetItem = item
				break
			}
		}
		if targetItem != nil {
			break
		}
	}

	if targetItem == nil {
		return nil, 0, 0, errors.New("物品不存在")
	}

	// 检查物品是否过期
	now := time.Now().Unix()
	if targetItem.ExpireTime > 0 && targetItem.ExpireTime < now {
		return nil, 0, 0, errors.New("物品已过期")
	}

	// 检查库存
	if targetItem.Stock > 0 && count > targetItem.Stock {
		return nil, 0, 0, errors.New("库存不足")
	}

	// 检查购买限制
	boughtCount := m.playerBuys[playerID][itemID]
	if targetItem.MaxBuyCount > 0 && boughtCount+count > targetItem.MaxBuyCount {
		return nil, 0, 0, errors.New("超出购买限制")
	}

	// 计算价格
	price := targetItem.CurrentPrice
	if targetItem.IsDiscount && targetItem.ExpireTime > now {
		price = int(float32(targetItem.OriginalPrice) * targetItem.DiscountRate)
	}
	totalPrice := price * count

	// 检查货币
	currency := m.playerCurrency[playerID]
	if targetItem.CurrencyType == 1 {
		// 金币
		if currency.Coins < totalPrice {
			return nil, 0, 0, errors.New("金币不足")
		}
		currency.Coins -= totalPrice
	} else {
		// 钻石
		if currency.Gems < totalPrice {
			return nil, 0, 0, errors.New("钻石不足")
		}
		currency.Gems -= totalPrice
	}

	// 更新购买记录
	m.playerBuys[playerID][itemID] += count

	// 减少库存
	if targetItem.Stock > 0 {
		targetItem.Stock -= count
	}

	// 创建购买的物品
	boughtItems := make([]Item, 0, count)
	for i := 0; i < count; i++ {
		item := Item{
			ID:         fmt.Sprintf("%s_item_%d_%d", playerID, itemID, time.Now().UnixNano()+int64(i)),
			ItemID:     int32(itemID),
			Count:      1,
			CreateTime: now,
			Attrs:      map[string]string{"source": "shop"},
		}
		boughtItems = append(boughtItems, item)
	}

	return boughtItems, totalPrice, targetItem.CurrencyType, nil
}

func (m *ShopManager) GetDiscountInfo(playerID string) []*DiscountInfo {
	// 示例折扣信息
	discounts := []*DiscountInfo{
		{
			ShopType:     1,
			DiscountRate: 0.9,
			StartTime:    time.Now().Unix() - 86400,
			EndTime:      time.Now().Unix() + 7*86400,
		},
		{
			ShopType:     2,
			DiscountRate: 0.7,
			StartTime:    time.Now().Unix() - 86400,
			EndTime:      time.Now().Unix() + 2*86400,
		},
	}

	return discounts
}

func (m *ShopManager) initShopItems() {
	// 普通商店
	m.shopItems[1] = []*ShopItem{
		{
			ItemID:        1001,
			OriginalPrice: 100,
			CurrentPrice:  100,
			CurrencyType:  1,
			Stock:         999,
			MaxBuyCount:   0,
			BoughtCount:   0,
			ExpireTime:    0,
			IsDiscount:    false,
			DiscountRate:  1.0,
		},
		{
			ItemID:        1002,
			OriginalPrice: 200,
			CurrentPrice:  200,
			CurrencyType:  1,
			Stock:         999,
			MaxBuyCount:   0,
			BoughtCount:   0,
			ExpireTime:    0,
			IsDiscount:    false,
			DiscountRate:  1.0,
		},
		{
			ItemID:        2001,
			OriginalPrice: 1000,
			CurrentPrice:  900,
			CurrencyType:  1,
			Stock:         50,
			MaxBuyCount:   0,
			BoughtCount:   0,
			ExpireTime:    0,
			IsDiscount:    true,
			DiscountRate:  0.9,
		},
	}

	// 限时商店
	weekLater := time.Now().Unix() + 7*86400
	m.shopItems[2] = []*ShopItem{
		{
			ItemID:        2005,
			OriginalPrice: 5000,
			CurrentPrice:  3500,
			CurrencyType:  1,
			Stock:         10,
			MaxBuyCount:   1,
			BoughtCount:   0,
			ExpireTime:    weekLater,
			IsDiscount:    true,
			DiscountRate:  0.7,
		},
		{
			ItemID:        3001,
			OriginalPrice: 100,
			CurrentPrice:  80,
			CurrencyType:  2,
			Stock:         20,
			MaxBuyCount:   5,
			BoughtCount:   0,
			ExpireTime:    weekLater,
			IsDiscount:    true,
			DiscountRate:  0.8,
		},
	}

	// 活动商店
	m.shopItems[3] = []*ShopItem{
		{
			ItemID:        4001,
			OriginalPrice: 1000,
			CurrentPrice:  1000,
			CurrencyType:  2,
			Stock:         99,
			MaxBuyCount:   0,
			BoughtCount:   0,
			ExpireTime:    0,
			IsDiscount:    false,
			DiscountRate:  1.0,
		},
	}
}
