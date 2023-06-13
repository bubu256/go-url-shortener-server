// Package server содержит структуры и методы grpc для работы grpc сервиса.
package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/url"

	"github.com/bubu256/go-url-shortener-server/config"
	"github.com/bubu256/go-url-shortener-server/internal/app/errorapp"
	"github.com/bubu256/go-url-shortener-server/internal/app/handlers"
	pb "github.com/bubu256/go-url-shortener-server/internal/app/proto"
	"github.com/bubu256/go-url-shortener-server/internal/app/schema"
	"github.com/bubu256/go-url-shortener-server/internal/app/shortener"
	"golang.org/x/exp/slices"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// Структура HandlerService хранит настройки для работы сервера и содержит gRPC методы
type HandlerService struct {
	pb.UnimplementedHandlerServiceServer
	service       *shortener.Shortener
	baseURL       string
	trustedSubnet *net.IPNet
	cfg           config.CfgServer
}

// New - возвращает ссылку на новую структуру handlerService, и *grpc.Server с подключенными перехватчиками
func New(service *shortener.Shortener, cfgServer config.CfgServer) (*HandlerService, *grpc.Server) {
	newHandlerService := HandlerService{
		baseURL:       cfgServer.BaseURL,
		trustedSubnet: handlers.ParseSubnetCIDR(cfgServer.TrustedSubnet),
		cfg:           cfgServer,
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
	return &pb.PingResponse{Success: true}, nil
}

// URLtoShort - Принимает полный URL и возвращает короткую ссылку.
// Если полный URL уже существует в базе возвращает ошибку и существующую короткую ссылку.
func (h *HandlerService) URLtoShort(ctx context.Context, req *pb.URLtoShortRequest) (*pb.URLtoShortResponse, error) {
	token := getToken(ctx)
	if token == "" {
		return nil, status.Error(codes.Unauthenticated, "пользователь не авторизован")
	}

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

// ShortToURL - возвращает полный URL по переданному короткому идентификаторы
func (h *HandlerService) ShortToURL(ctx context.Context, req *pb.ShortToURLRequest) (*pb.ShortToURLResponse, error) {
	fullURL, err := h.service.GetURL(req.ShortKey)
	if err != nil {
		if errors.Is(err, errorapp.ErrorPageNotAvailable) {
			return nil, status.Errorf(codes.NotFound, "ресурс больше не доступен %v;", err)
		}
		return nil, status.Errorf(codes.NotFound, "ресурс отсутствует %v;", err)
	}
	return &pb.ShortToURLResponse{FullUrl: fullURL}, nil
}

// APIShortenBatch - записывает переданные сокращенные идентификаторы и полные URL в хранилище.
func (h *HandlerService) APIShortenBatch(ctx context.Context, req *pb.APIShortenBatchRequest) (*pb.APIShortenBatchResponse, error) {
	token := getToken(ctx)
	if token == "" {
		return nil, status.Error(codes.Unauthenticated, "пользователь не авторизован;")
	}
	// формируем батч для обработки
	batch := make(schema.APIShortenBatchInput, len(req.Urls))
	for i, elem := range req.Urls {
		batch[i].CorrelationID = elem.CorrelationId
		batch[i].OriginalURL = elem.OriginalUrl
	}
	// получаем идентификаторы ссылок записанные в базу
	shortKeys, err := h.service.SetBatchURLs(batch, token)
	if err != nil {
		return nil, status.Error(codes.Internal, "ошибка при добавлении batch ссылок;")
	}
	result := make([]*pb.ShortURLMapping, len(shortKeys))
	for _, key := range shortKeys {
		shortURL, err := h.createLink(key)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "ошибка при формировании короткой ссылки %v;", err)
		}
		result = append(result, &pb.ShortURLMapping{CorrelationId: key, ShortUrl: shortURL})
	}
	return &pb.APIShortenBatchResponse{ShortUrls: result}, nil
}

// APIUserAllURLs - возвращает все URL пользователя
func (h *HandlerService) APIUserAllURLs(ctx context.Context, req *pb.APIUserAllURLsRequest) (*pb.APIUserAllURLsResponse, error) {
	token := getToken(ctx)
	if token == "" {
		return nil, status.Error(codes.Unauthenticated, "пользователь не авторизован;")
	}
	allURLs := h.service.GetAllURLs(token)
	result := make([]*pb.URLMapping, 0)
	for key, URL := range allURLs {
		result = append(result, &pb.URLMapping{CorrelationId: key, OriginalUrl: URL})
	}
	return &pb.APIUserAllURLsResponse{Urls: result}, nil
}

// APIDeleteUrls - принимает запрос на удаление URLs. Удаление возможно только для URLs добавленных пользователем.
// метод только принимает запрос, удаление может произойти позже.
func (h *HandlerService) APIDeleteUrls(ctx context.Context, req *pb.APIDeleteUrlsRequest) (*pb.APIDeleteUrlsResponse, error) {
	token := getToken(ctx)
	if token == "" {
		return nil, status.Error(codes.Unauthenticated, "пользователь не авторизован;")
	}
	go h.service.DeleteBatch(req.Urls, token)

	return &pb.APIDeleteUrlsResponse{Success: true}, nil
}

// APIInternalStats - возвращает статистику сервера
func (h *HandlerService) APIInternalStats(ctx context.Context, req *pb.APIInternalStatsRequest) (*pb.APIInternalStatsResponse, error) {
	p, _ := peer.FromContext(ctx)
	remoteAddr := p.Addr.String()
	if !h.isTrustedSubnet(remoteAddr) {
		return nil, status.Error(codes.PermissionDenied, "метод не доступен")
	}
	// получаем статистику
	stats, err := h.service.GetStatsStorage()
	if err != nil {
		return nil, status.Error(codes.Internal, "ошибка при получении статистики по серверу;")
	}
	return &pb.APIInternalStatsResponse{Users: int32(stats.Users), Urls: int32(stats.URLs)}, nil
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
	// Исключаем методы TokenHandler, ShortToURL из проверки токена и
	log.Println(info.FullMethod)
	if slices.Contains(
		[]string{"/server.HandlerService/TokenHandler", "/server.HandlerService/ShortToURL"},
		info.FullMethod,
	) {
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

// isTrustedSubnet - проверяет входит ли IP-адрес в доверительную подсеть сервера
func (h *HandlerService) isTrustedSubnet(remoteAddr string) bool {
	if h.trustedSubnet == nil {
		return false
	}
	// Парсим IP-адрес из Request.RemoteAddr
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		log.Println("Error splitting host and port:", err)
		// return false // возможно строка не содержит порт, проверяем дальше
		host = remoteAddr
	}
	IP := net.ParseIP(host)
	if IP == nil {
		log.Println("Error parsing IP address:", err)
		return false
	}

	// Проверяем входит ли IP-адрес в доверительную подсеть
	return h.trustedSubnet.Contains(IP)
}

// getToken - возвращает токен из контекста, если он есть.
func getToken(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	// получение токена
	values := md.Get("token")
	if len(values) == 0 {
		return ""
	}
	return values[0]
}
