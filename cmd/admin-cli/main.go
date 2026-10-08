package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	pb "github.com/MAMUER/project/api/gen/user"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"

	"github.com/MAMUER/project/internal/auth/claims"
	"github.com/MAMUER/project/internal/auth/jwt"
	grpctls "github.com/MAMUER/project/internal/grpc"
)

const (
	defaultUserServiceAddr = "localhost:50051"
	defaultTimeout         = 10 * time.Second

	requesterIDFlag             = "requester-id"
	requesterIDDescription      = "ID администратора (опционально, если задан ADMIN_CLI_JWT)"
	requesterAndUserIDRequired  = "Ошибка: --requester-id и --id обязательны"
)

func toInt32(v int) int32 {
	if v < math.MinInt32 || v > math.MaxInt32 {
		panic(fmt.Sprintf("admin-cli: value %d overflows int32", v))
	}
	return int32(v)
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	args := os.Args[2:]

	switch command {
	case "create-invite":
		createInvite(args)
	case "list-invites":
		listInvites(args)
	case "revoke-invite":
		revokeInvite(args)
	case "list-users":
		listUsers(args)
	case "delete-user":
		deleteUser(args)
	case "ban-user":
		banUser(args)
	case "unban-user":
		unbanUser(args)
	default:
		fmt.Fprintf(os.Stderr, "Неизвестная команда: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("FitPulse — Административный CLI")
	fmt.Println()
	fmt.Println("Использование: admin-cli <команда> [параметры]")
	fmt.Println()
	fmt.Println("Команды:")
	fmt.Println("  create-invite --role <роль> [--max-uses <n>]")
	fmt.Println("  list-invites")
	fmt.Println("  revoke-invite --code <код>")
	fmt.Println("  list-users")
	fmt.Println("  delete-user --id <user_id>")
	fmt.Println("  ban-user --id <user_id>")
	fmt.Println("  unban-user --id <user_id>")
	fmt.Println()
	fmt.Println("Переменные окружения:")
	fmt.Println("  USER_SERVICE_ADDR  gRPC адрес user-service (по умолчанию: localhost:50051)")
	fmt.Println("  GRPC_TLS_CERT_FILE Сертификат сервера для mTLS (опционально)")
	fmt.Println("  GRPC_TLS_KEY_FILE  Приватный ключ сервера для mTLS (опционально)")
	fmt.Println("  GRPC_TLS_CA_FILE   CA сертификат для проверки сервера (опционально)")
	fmt.Println("  ADMIN_CLI_JWT      JWT access token для авторизации (опционально)")
}

func getUserServiceAddr() string {
	if addr := os.Getenv("USER_SERVICE_ADDR"); addr != "" {
		return addr
	}
	return defaultUserServiceAddr
}

func getTimeout() time.Duration {
	if timeoutStr := os.Getenv("ADMIN_CLI_TIMEOUT"); timeoutStr != "" {
		if d, err := time.ParseDuration(timeoutStr); err == nil {
			return d
		}
	}
	return defaultTimeout
}

func validateJWT() (*claims.Claims, error) {
	jwtToken := os.Getenv("ADMIN_CLI_JWT")
	if jwtToken == "" {
		return nil, nil
	}

	publicKeyPEM := os.Getenv("JWT_PUBLIC_KEY_PEM")
	if publicKeyPEM == "" {
		return nil, errors.New("JWT_PUBLIC_KEY_PEM не задан, невозможно проверить токен")
	}

	claims, err := jwt.ValidateAccessToken(jwtToken, publicKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("JWT токен невалиден: %w", err)
	}

	if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now()) {
		return nil, errors.New("JWT токен истёк")
	}

	return claims, nil
}

func connectUserService() (*grpc.ClientConn, pb.UserServiceClient, error) {
	addr := getUserServiceAddr()

	if _, err := validateJWT(); err != nil {
		return nil, nil, fmt.Errorf("ошибка валидации JWT: %w", err)
	}

	dialOpts := []grpc.DialOption{
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                30 * time.Second,
			Timeout:             5 * time.Second,
			PermitWithoutStream: true,
		}),
		grpc.WithBlock(),
	}

	if jwtToken := os.Getenv("ADMIN_CLI_JWT"); jwtToken != "" {
		dialOpts = append(dialOpts, grpc.WithPerRPCCredentials(&jwtCredentials{token: jwtToken}))
	}

	creds, err := grpctls.GetClientTLSCredentials()
	if err != nil {
		return nil, nil, fmt.Errorf("ошибка загрузки TLS credentials: %w", err)
	}
	if creds == nil {
		return nil, nil, errors.New("TLS не настроен: задайте GRPC_TLS_CA_FILE для подключения к user-service")
	}
	dialOpts = append(dialOpts, grpc.WithTransportCredentials(creds))

	conn, err := grpc.NewClient(addr, dialOpts...)
	if err != nil {
		return nil, nil, fmt.Errorf("Не удалось подключиться к user-service по адресу %s: %w", addr, err)
	}

	client := pb.NewUserServiceClient(conn)
	return conn, client, nil
}

type jwtCredentials struct {
	token string
}

func (c *jwtCredentials) RequireTransportSecurity() bool {
	return true
}

func (c *jwtCredentials) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	return map[string]string{
		"authorization": "Bearer " + c.token,
	}, nil
}

func createInvite(args []string) {
	fs := flag.NewFlagSet("create-invite", flag.ExitOnError)
	role := fs.String("role", "client", "Роль для кода приглашения (client|admin)")
	maxUses := fs.Int("max-uses", 1, "Максимальное количество использований")
	fs.Parse(args)

	if *role != "client" && *role != "admin" {
		fmt.Fprintln(os.Stderr, "Ошибка: роль должна быть 'client' или 'admin'")
		os.Exit(1)
	}
	if *maxUses < 1 || *maxUses > 100 {
		fmt.Fprintln(os.Stderr, "Ошибка: --max-uses должно быть от 1 до 100")
		os.Exit(1)
	}

	conn, client, err := connectUserService()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), getTimeout())
	defer cancel()

	resp, err := client.AdminCreateInvite(ctx, &pb.AdminCreateInviteRequest{
		Role:    *role,
		MaxUses: toInt32(*maxUses),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка: %v\n", status.Convert(err).Message())
		os.Exit(1)
	}

	fmt.Printf("Код приглашения создан:\n")
	fmt.Printf("  Код:      %s\n", resp.Code)
	fmt.Printf("  Роль:      %s\n", resp.Role)
	fmt.Printf("  Макс. использований:  %d\n", resp.MaxUses)
	fmt.Printf("  Ссылка: %s\n", resp.InviteUrl)
	fmt.Printf("  Создан: %s\n", resp.CreatedAt)
}

func listInvites(args []string) {
	fs := flag.NewFlagSet("list-invites", flag.ExitOnError)
	page := fs.Int("page", 1, "Номер страницы")
	pageSize := fs.Int("page-size", 20, "Размер страницы")
	fs.Parse(args)

	if *page < 1 {
		*page = 1
	}
	if *pageSize < 1 || *pageSize > 100 {
		*pageSize = 20
	}

	conn, client, err := connectUserService()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), getTimeout())
	defer cancel()

	resp, err := client.AdminListInvites(ctx, &pb.AdminListInvitesRequest{
		Page:     toInt32(*page),
		PageSize: toInt32(*pageSize),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка: %v\n", status.Convert(err).Message())
		os.Exit(1)
	}

	fmt.Printf("Всего приглашений: %d\n\n", resp.Total)
	fmt.Printf("%-20s %-10s %-10s %-10s %-10s %-10s\n", "КОД", "РОЛЬ", "МАКС", "ИСПОЛЬЗОВАНО", "АКТИВНО", "СОЗДАНО")
	fmt.Println(strings.Repeat("-", 80))
	for _, inv := range resp.Invites {
		active := "Нет"
		if inv.IsActive {
			active = "Да"
		}
		fmt.Printf("%-20s %-10s %-10d %-10d %-10s %-10s\n",
			inv.Code, inv.Role, inv.MaxUses, inv.UsedCount, active, inv.CreatedAt)
	}
}

func revokeInvite(args []string) {
	fs := flag.NewFlagSet("revoke-invite", flag.ExitOnError)
	code := fs.String("code", "", "Код приглашения для отзыва")
	fs.Parse(args)

	if *code == "" {
		fmt.Fprintln(os.Stderr, "Ошибка: --code обязателен")
		os.Exit(1)
	}

	conn, client, err := connectUserService()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), getTimeout())
	defer cancel()

	resp, err := client.AdminRevokeInvite(ctx, &pb.AdminRevokeInviteRequest{
		Code: *code,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка: %v\n", status.Convert(err).Message())
		os.Exit(1)
	}

	if resp.Success {
		fmt.Printf("Код приглашения %s успешно отозван\n", *code)
	} else {
		fmt.Fprintf(os.Stderr, "Не удалось отозвать приглашение: %s\n", resp.Message)
		os.Exit(1)
	}
}

func listUsers(args []string) {
	fs := flag.NewFlagSet("list-users", flag.ExitOnError)
	requesterID := fs.String(requesterIDFlag, "", requesterIDDescription)
	page := fs.Int("page", 1, "Номер страницы")
	pageSize := fs.Int("page-size", 20, "Размер страницы")
	role := fs.String("role", "", "Фильтр по роли (client|admin)")
	fs.Parse(args)

	requesterIDValue := *requesterID
	if requesterIDValue == "" {
		if claims, err := validateJWT(); err != nil {
			fmt.Fprintf(os.Stderr, "Ошибка валидации JWT: %v\n", err)
			os.Exit(1)
		} else if claims != nil {
			requesterIDValue = claims.UserID
		}
	}
	if requesterIDValue == "" {
		fmt.Fprintln(os.Stderr, "Ошибка: --requester-id обязателен или задайте ADMIN_CLI_JWT")
		os.Exit(1)
	}
	if *page < 1 {
		*page = 1
	}
	if *pageSize < 1 || *pageSize > 100 {
		*pageSize = 20
	}

	conn, client, err := connectUserService()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), getTimeout())
	defer cancel()

	resp, err := client.ListUsers(ctx, &pb.ListUsersRequest{
		RequesterUserId: requesterIDValue,
		Page:            toInt32(*page),
		PageSize:        toInt32(*pageSize),
		Role:            *role,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка: %v\n", status.Convert(err).Message())
		os.Exit(1)
	}

	fmt.Printf("Всего пользователей: %d\n\n", resp.Total)
	fmt.Printf("%-36s %-20s %-10s %-10s %-20s\n", "ID ПОЛЬЗОВАТЕЛЯ", "EMAIL", "РОЛЬ", "ПОДТВ.", "СОЗДАНО")
	fmt.Println(strings.Repeat("-", 100))
	for _, u := range resp.Users {
		verified := "Нет"
		if u.EmailConfirmed {
			verified = "Да"
		}
		email := obfuscateEmail(u.Email)
		fmt.Printf("%-36s %-20s %-10s %-10s %-20s\n",
			u.UserId, email, u.Role, verified, u.CreatedAt)
	}
}

func deleteUser(args []string) {
	fs := flag.NewFlagSet("delete-user", flag.ExitOnError)
	requesterID := fs.String(requesterIDFlag, "", requesterIDDescription)
	userID := fs.String("id", "", "ID пользователя для удаления")
	fs.Parse(args)

	requesterIDValue := *requesterID
	if requesterIDValue == "" {
		if claims, err := validateJWT(); err != nil {
			fmt.Fprintf(os.Stderr, "Ошибка валидации JWT: %v\n", err)
			os.Exit(1)
		} else if claims != nil {
			requesterIDValue = claims.UserID
		}
	}
	if requesterIDValue == "" || *userID == "" {
		fmt.Fprintln(os.Stderr, requesterAndUserIDRequired)
		os.Exit(1)
	}

	conn, client, err := connectUserService()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), getTimeout())
	defer cancel()

	resp, err := client.AdminDeleteUser(ctx, &pb.AdminDeleteUserRequest{
		RequesterUserId: requesterIDValue,
		UserId:          *userID,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка: %v\n", status.Convert(err).Message())
		os.Exit(1)
	}

	if resp.Success {
		fmt.Printf("Пользователь %s успешно удалён\n", *userID)
	} else {
		fmt.Fprintf(os.Stderr, "Не удалось удалить пользователя: %s\n", resp.Message)
		os.Exit(1)
	}
}

func banUser(args []string) {
	fs := flag.NewFlagSet("ban-user", flag.ExitOnError)
	requesterID := fs.String(requesterIDFlag, "", requesterIDDescription)
	userID := fs.String("id", "", "ID пользователя для бана")
	fs.Parse(args)

	requesterIDValue := *requesterID
	if requesterIDValue == "" {
		if claims, err := validateJWT(); err != nil {
			fmt.Fprintf(os.Stderr, "Ошибка валидации JWT: %v\n", err)
			os.Exit(1)
		} else if claims != nil {
			requesterIDValue = claims.UserID
		}
	}
	if requesterIDValue == "" || *userID == "" {
		fmt.Fprintln(os.Stderr, requesterAndUserIDRequired)
		os.Exit(1)
	}

	conn, client, err := connectUserService()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), getTimeout())
	defer cancel()

	resp, err := client.AdminBanUser(ctx, &pb.AdminBanUserRequest{
		RequesterUserId: requesterIDValue,
		UserId:          *userID,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка: %v\n", status.Convert(err).Message())
		os.Exit(1)
	}

	if resp.Success {
		fmt.Printf("Пользователь %s успешно забанен\n", *userID)
	} else {
		fmt.Fprintf(os.Stderr, "Не удалось забанить пользователя: %s\n", resp.Message)
		os.Exit(1)
	}
}

func unbanUser(args []string) {
	fs := flag.NewFlagSet("unban-user", flag.ExitOnError)
	requesterID := fs.String(requesterIDFlag, "", requesterIDDescription)
	userID := fs.String("id", "", "ID пользователя для разбана")
	fs.Parse(args)

	requesterIDValue := *requesterID
	if requesterIDValue == "" {
		if claims, err := validateJWT(); err != nil {
			fmt.Fprintf(os.Stderr, "Ошибка валидации JWT: %v\n", err)
			os.Exit(1)
		} else if claims != nil {
			requesterIDValue = claims.UserID
		}
	}
	if requesterIDValue == "" || *userID == "" {
		fmt.Fprintln(os.Stderr, requesterAndUserIDRequired)
		os.Exit(1)
	}

	conn, client, err := connectUserService()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), getTimeout())
	defer cancel()

	resp, err := client.AdminUnbanUser(ctx, &pb.AdminUnbanUserRequest{
		RequesterUserId: requesterIDValue,
		UserId:          *userID,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка: %v\n", status.Convert(err).Message())
		os.Exit(1)
	}

	if resp.Success {
		fmt.Printf("Пользователь %s успешно разбанен\n", *userID)
	} else {
		fmt.Fprintf(os.Stderr, "Не удалось разбанить пользователя: %s\n", resp.Message)
		os.Exit(1)
	}
}

func obfuscateEmail(email string) string {
	if email == "" {
		return ""
	}
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return email
	}
	local := parts[0]
	domain := parts[1]
	if len(local) > 2 {
		local = local[:2] + "***"
	} else if len(local) > 0 {
		local = local[:1] + "***"
	}
	return local + "@" + domain
}
