package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/url"

	"github.com/bubu256/go-url-shortener-server/config"
	"github.com/bubu256/go-url-shortener-server/internal/app/errorapp"
	pb "github.com/bubu256/go-url-shortener-server/internal/app/proto"
	"github.com/bubu256/go-url-shortener-server/internal/app/shortener"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type HandlerService struct {
	pb.UnimplementedHandlerServiceServer
	service *shortener.Shortener
	baseURL string
	// trustedSubnet *net.IPNet
	cfg config.CfgServer
}

// New - возвращает ссылку на новую структуру handlerService, и *grpc.Server с подключенными перехватчиками
func New(service *shortener.Shortener, cfgServer config.CfgServer) (*HandlerService, *grpc.Server) {
	newHandlerService := HandlerService{
		baseURL: cfgServer.BaseURL,
		// trustedSubnet: ParseSubnetCIDR(cfgServer.TrustedSubnet),
		cfg: cfgServer,
	}
	return &newHandlerService, grpc.NewServer(
		grpc.UnaryInterceptor(newHandlerService.tokenInterceptor),
	)
}

// Ping - метод вВыполняет проверку соединения с БД.
// возвращает пустую структуру и nil в случае успеха
func (h *HandlerService) Ping(ctx context.Context, req *pb.PingRequest) (*pb.PingResponse, error) {
	err := h.service.PingDB()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "ошибка подключения к БД; %V", err)
	}
	return &pb.PingResponse{}, nil
}

// URLtoShort - Принимает полный URL и возвращает короткую ссылку.
// Если полный URL уже существует в базе возвращает ошибку и существующую короткую ссылку.
func (h *HandlerService) URLtoShort(ctx context.Context, req *pb.URLtoShortRequest) (*pb.URLtoShortResponse, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Internal, "failed to get metadata from context;")
	}
	// получение токена
	values := md.Get("token")
	if len(values) == 0 {
		return nil, status.Error(codes.Internal, "произошла потеря токена;")
	}
	token := values[0]
	// получаем короткий идентификатор ссылки
	shortKey, err := h.service.CreateShortKey(req.Url, token)
	var errDuplicate *errorapp.URLDuplicateError
	if errors.As(err, &errDuplicate) {
		// если ошибка дубликации урл
		shortURL, err := h.createLink(errDuplicate.ExistsKey)
		if err != nil {
			return nil, status.Error(codes.Internal, fmt.Errorf("ошибка при сборе короткой ссылки %v; %w", err, errDuplicate).Error())
		}
		return &pb.URLtoShortResponse{ShortUrl: shortURL}, status.Errorf(codes.InvalidArgument, "найден дубликат; %v", errDuplicate)
	} else if err != nil {
		return nil, status.Errorf(codes.Internal, "ошибка при создании короткого ключа %v;", err)
	}
	// собираем сокращенную ссылку
	shortURL, err := h.createLink(shortKey)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "ошибка при сборе короткой ссылки %v;", err)
	}
	return &pb.URLtoShortResponse{ShortUrl: shortURL}, nil
}

func (h *HandlerService) ShortToURL(ctx context.Context, req *pb.ShortToURLRequest) (*pb.ShortToURLResponse, error) {
	// TODO: Implement logic for ShortToURL handler
	return &pb.ShortToURLResponse{}, nil
}

func (h *HandlerService) APIShortenBatch(ctx context.Context, req *pb.APIShortenBatchRequest) (*pb.APIShortenBatchResponse, error) {
	// TODO: Implement logic for APIShortenBatch handler
	return &pb.APIShortenBatchResponse{}, nil
}

func (h *HandlerService) APIShorten(ctx context.Context, req *pb.APIShortenRequest) (*pb.APIShortenResponse, error) {
	// TODO: Implement logic for APIShorten handler
	return &pb.APIShortenResponse{}, nil
}

func (h *HandlerService) APIUserAllURLs(ctx context.Context, req *pb.APIUserAllURLsRequest) (*pb.APIUserAllURLsResponse, error) {
	// TODO: Implement logic for APIUserAllURLs handler
	return &pb.APIUserAllURLsResponse{}, nil
}

func (h *HandlerService) APIDeleteUrls(ctx context.Context, req *pb.APIDeleteUrlsRequest) (*pb.APIDeleteUrlsResponse, error) {
	// TODO: Implement logic for APIDeleteUrls handler
	return &pb.APIDeleteUrlsResponse{}, nil
}

func (h *HandlerService) APIInternalStats(ctx context.Context, req *pb.APIInternalStatsRequest) (*pb.APIInternalStatsResponse, error) {
	// TODO: Implement logic for APIInternalStats handler
	return &pb.APIInternalStatsResponse{}, nil
}

// TokenHandler - выдает токен пользователю
func (h *HandlerService) TokenHandler(ctx context.Context, req *pb.TokenHandlerRequest) (*pb.TokenHandlerResponse, error) {
	// если токен во входящей структуре верный просто возвращаем его обратно
	if req.Token != "" && h.service.CheckToken(req.Token) {
		return &pb.TokenHandlerResponse{Token: req.Token}, nil
	}
	// иначе выдаем новый токен
	newToken, err := h.service.GenerateNewToken()
	if err != nil {
		return nil, status.Error(codes.Internal, "ошибка при создании токена;")
	}
	return &pb.TokenHandlerResponse{Token: newToken}, nil
}

// createLink - метод создает короткую ссылку на основе ключа
func (h *HandlerService) createLink(shortKey string) (string, error) {
	return url.JoinPath(h.baseURL, shortKey)
}

// tokenInterceptor - перехватчик проверяет наличие и валидность токена
func (h *HandlerService) tokenInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	// Исключаем метод TokenHandler из проверки токена
	log.Println(info.FullMethod)
	if info.FullMethod == "/server.HandlerService/TokenHandler" {
		return handler(ctx, req)
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Internal, "Failed to get metadata from context")
	}

	// Проверка наличия токена в метаданных
	values := md.Get("token")
	if len(values) == 0 {
		return nil, status.Error(codes.Unauthenticated, "Token is missing")
	}
	token := values[0]
	if !h.service.CheckToken(token) {
		return nil, status.Error(codes.Unauthenticated, "Token is invalid")
	}
	return handler(ctx, req)
}
