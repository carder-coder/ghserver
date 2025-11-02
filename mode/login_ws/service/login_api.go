package server

import (
	"context"
	loginpb "ghserver/proto/pb"
	RPC "ghserver/proto/rpc_ctl"

	code "ghserver/define/code"

	"github.com/dobyte/due/component/http/v2"
	"github.com/dobyte/due/v2/log"
	"github.com/go-playground/validator/v10"
)

type LoginApi struct {
	proxy    *http.Proxy
	validate *validator.Validate
}

func NewLoginAPI(proxy *http.Proxy) *LoginApi {
	return &LoginApi{
		proxy:    proxy,
		validate: validator.New(),
	}
}
func (a *LoginApi) Close() error {
	return nil
}
func (a *LoginApi) Init() {
	// 路由器
	router := a.proxy.Router()
	// 登录
	router.Post("/login", a.Login)
	// 注册
	router.Post("/register", a.Register)
}

type LoginReq struct {
	Account  string `json:"account" validate:"required"`  // 账号
	Password string `json:"password" validate:"required"` // 密码
}

type LoginRes struct {
	Gate  string `json:"gate"`  // 网关
	Token string `json:"token"` // Token
}

type RegisterReq struct {
	Account  string `json:"account" validate:"required"`  // 账号
	Nickname string `json:"nickname" validate:"required"` // 昵称
	Password string `json:"password" validate:"required"` // 密码
}

// Login 登录
// @Summary 登录
// @Tags 登录
// @Schemes
// @Accept json
// @Produce json
// @Param request body LoginReq true "请求参数"
// @Response 200 {object} http.Resp{Data=LoginRes} "响应参数"
// @Router /login [post]
func (a *LoginApi) Login(ctx http.Context) error {
	req := &LoginReq{}

	if err := ctx.Bind().JSON(req); err != nil {
		return ctx.Failure(code.InvalidArgument)
	}

	if err := a.validate.Struct(req); err != nil {
		return ctx.Failure(code.InvalidArgument)
	}

	client, err := RPC.NewRpcClient(a.proxy.NewMeshClient, RPC.ServiceTypeLogin)
	if err != nil {
		log.Errorf("create client failed: %v", err)
		return ctx.Failure(code.InternalError)
	}

	reply, err := client.(loginpb.LoginServiceClient).Login(context.Background(), &loginpb.LoginRequest{
		Account:  req.Account,
		Password: req.Password,
		DeviceId: ctx.IP(),
	})
	if err != nil {
		return ctx.Failure(err)
	}

	return ctx.Success(&loginpb.LoginResponse{
		Code:          reply.Code,
		Message:       reply.Message,
		Token:         reply.Token,
		Player:        reply.Player,
		QueuePosition: reply.QueuePosition,
	})
}

// Register 注册
// @Summary 注册
// @Tags 注册
// @Schemes
// @Accept json
// @Produce json
// @Param request body RegisterReq true "请求参数"
// @Response 200 {object} http.Resp{} "响应参数"
// @Router /register [post]
func (a *LoginApi) Register(ctx http.Context) error {
	req := &RegisterReq{}

	if err := ctx.Bind().JSON(req); err != nil {
		return ctx.Failure(code.InvalidArgument)
	}

	if err := a.validate.Struct(req); err != nil {
		return ctx.Failure(code.InvalidArgument)
	}

	client, err := RPC.NewRpcClient(a.proxy.NewMeshClient, RPC.ServiceTypeLogin)
	if err != nil {
		log.Errorf("create client failed: %v", err)
		return ctx.Failure(code.InternalError)
	}

	_, err = client.(loginpb.LoginServiceClient).Register(context.Background(), &loginpb.RegisterRequest{
		Account:  req.Account,
		Password: req.Password,
		Nickname: req.Nickname,
		ClientIP: ctx.IP(),
	})
	if err != nil {
		return ctx.Failure(err)
	}

	return ctx.Success(&loginpb.RegisterResponse{Code: code.OK, Message: "注册成功"})
}
