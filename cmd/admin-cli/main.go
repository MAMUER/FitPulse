package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"

	pb "github.com/MAMUER/project/api/gen/user"
)

const (
	defaultUserServiceAddr = "localhost:50051"
	defaultTimeout         = 10 * time.Second
)

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
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("FitPulse Admin CLI")
	fmt.Println()
	fmt.Println("Usage: admin-cli <command> [options]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  create-invite --role <role> [--max-uses <n>]")
	fmt.Println("  list-invites")
	fmt.Println("  revoke-invite --code <code>")
	fmt.Println("  list-users")
	fmt.Println("  delete-user --id <user_id>")
	fmt.Println("  ban-user --id <user_id>")
	fmt.Println("  unban-user --id <user_id>")
	fmt.Println()
	fmt.Println("Environment:")
	fmt.Println("  USER_SERVICE_ADDR  gRPC address of user-service (default: localhost:50051)")
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

func connectUserService() (*grpc.ClientConn, pb.UserServiceClient, error) {
	addr := getUserServiceAddr()
	timeout := getTimeout()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	kac := keepalive.ClientParameters{
		Time:                30 * time.Second,
		Timeout:             5 * time.Second,
		PermitWithoutStream: true,
	}

	conn, err := grpc.DialContext(
		ctx,
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithKeepaliveParams(kac),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to user-service at %s: %w", addr, err)
	}

	client := pb.NewUserServiceClient(conn)
	return conn, client, nil
}

func createInvite(args []string) {
	fs := flag.NewFlagSet("create-invite", flag.ExitOnError)
	role := fs.String("role", "client", "Role for invite code (client|admin)")
	maxUses := fs.Int("max-uses", 1, "Maximum number of uses")
	fs.Parse(args)

	if *role != "client" && *role != "admin" {
		fmt.Fprintln(os.Stderr, "Error: role must be 'client' or 'admin'")
		os.Exit(1)
	}
	if *maxUses < 1 || *maxUses > 100 {
		fmt.Fprintln(os.Stderr, "Error: max-uses must be between 1 and 100")
		os.Exit(1)
	}

	conn, client, err := connectUserService()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), getTimeout())
	defer cancel()

	resp, err := client.AdminCreateInvite(ctx, &pb.AdminCreateInviteRequest{
		Role:    *role,
		MaxUses: int32(*maxUses),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", status.Convert(err).Message())
		os.Exit(1)
	}

	fmt.Printf("Invite code created:\n")
	fmt.Printf("  Code:      %s\n", resp.Code)
	fmt.Printf("  Role:      %s\n", resp.Role)
	fmt.Printf("  Max uses:  %d\n", resp.MaxUses)
	fmt.Printf("  Invite URL: %s\n", resp.InviteUrl)
	fmt.Printf("  Created at: %s\n", resp.CreatedAt)
}

func listInvites(args []string) {
	fs := flag.NewFlagSet("list-invites", flag.ExitOnError)
	page := fs.Int("page", 1, "Page number")
	pageSize := fs.Int("page-size", 20, "Page size")
	fs.Parse(args)

	if *page < 1 {
		*page = 1
	}
	if *pageSize < 1 || *pageSize > 100 {
		*pageSize = 20
	}

	conn, client, err := connectUserService()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), getTimeout())
	defer cancel()

	resp, err := client.AdminListInvites(ctx, &pb.AdminListInvitesRequest{
		Page:     int32(*page),
		PageSize: int32(*pageSize),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", status.Convert(err).Message())
		os.Exit(1)
	}

	fmt.Printf("Total invites: %d\n\n", resp.Total)
	fmt.Printf("%-20s %-10s %-10s %-10s %-10s %-10s\n", "CODE", "ROLE", "MAX USES", "USED", "ACTIVE", "CREATED AT")
	fmt.Println(strings.Repeat("-", 80))
	for _, inv := range resp.Invites {
		active := "No"
		if inv.IsActive {
			active = "Yes"
		}
		fmt.Printf("%-20s %-10s %-10d %-10d %-10s %-10s\n",
			inv.Code, inv.Role, inv.MaxUses, inv.UsedCount, active, inv.CreatedAt)
	}
}

func revokeInvite(args []string) {
	fs := flag.NewFlagSet("revoke-invite", flag.ExitOnError)
	code := fs.String("code", "", "Invite code to revoke")
	fs.Parse(args)

	if *code == "" {
		fmt.Fprintln(os.Stderr, "Error: --code is required")
		os.Exit(1)
	}

	conn, client, err := connectUserService()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), getTimeout())
	defer cancel()

	resp, err := client.AdminRevokeInvite(ctx, &pb.AdminRevokeInviteRequest{
		Code: *code,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", status.Convert(err).Message())
		os.Exit(1)
	}

	if resp.Success {
		fmt.Printf("Invite code %s revoked successfully\n", *code)
	} else {
		fmt.Fprintf(os.Stderr, "Failed to revoke invite: %s\n", resp.Message)
		os.Exit(1)
	}
}

func listUsers(args []string) {
	fs := flag.NewFlagSet("list-users", flag.ExitOnError)
	requesterID := fs.String("requester-id", "", "Admin user ID")
	page := fs.Int("page", 1, "Page number")
	pageSize := fs.Int("page-size", 20, "Page size")
	role := fs.String("role", "", "Filter by role (client|admin)")
	fs.Parse(args)

	if *requesterID == "" {
		fmt.Fprintln(os.Stderr, "Error: --requester-id is required")
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
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), getTimeout())
	defer cancel()

	resp, err := client.ListUsers(ctx, &pb.ListUsersRequest{
		RequesterUserId: *requesterID,
		Page:            int32(*page),
		PageSize:        int32(*pageSize),
		Role:            *role,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", status.Convert(err).Message())
		os.Exit(1)
	}

	fmt.Printf("Total users: %d\n\n", resp.Total)
	fmt.Printf("%-36s %-20s %-10s %-10s %-20s\n", "USER ID", "EMAIL", "ROLE", "VERIFIED", "CREATED AT")
	fmt.Println(strings.Repeat("-", 100))
	for _, u := range resp.Users {
		verified := "No"
		if u.EmailConfirmed {
			verified = "Yes"
		}
		email := obfuscateEmail(u.Email)
		fmt.Printf("%-36s %-20s %-10s %-10s %-20s\n",
			u.UserId, email, u.Role, verified, u.CreatedAt)
	}
}

func deleteUser(args []string) {
	fs := flag.NewFlagSet("delete-user", flag.ExitOnError)
	requesterID := fs.String("requester-id", "", "Admin user ID")
	userID := fs.String("id", "", "User ID to delete")
	fs.Parse(args)

	if *requesterID == "" || *userID == "" {
		fmt.Fprintln(os.Stderr, "Error: --requester-id and --id are required")
		os.Exit(1)
	}

	conn, client, err := connectUserService()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), getTimeout())
	defer cancel()

	resp, err := client.AdminDeleteUser(ctx, &pb.AdminDeleteUserRequest{
		RequesterUserId: *requesterID,
		UserId:          *userID,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", status.Convert(err).Message())
		os.Exit(1)
	}

	if resp.Success {
		fmt.Printf("User %s deleted successfully\n", *userID)
	} else {
		fmt.Fprintf(os.Stderr, "Failed to delete user: %s\n", resp.Message)
		os.Exit(1)
	}
}

func banUser(args []string) {
	fs := flag.NewFlagSet("ban-user", flag.ExitOnError)
	requesterID := fs.String("requester-id", "", "Admin user ID")
	userID := fs.String("id", "", "User ID to ban")
	fs.Parse(args)

	if *requesterID == "" || *userID == "" {
		fmt.Fprintln(os.Stderr, "Error: --requester-id and --id are required")
		os.Exit(1)
	}

	conn, client, err := connectUserService()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), getTimeout())
	defer cancel()

	resp, err := client.AdminBanUser(ctx, &pb.AdminBanUserRequest{
		RequesterUserId: *requesterID,
		UserId:          *userID,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", status.Convert(err).Message())
		os.Exit(1)
	}

	if resp.Success {
		fmt.Printf("User %s banned successfully\n", *userID)
	} else {
		fmt.Fprintf(os.Stderr, "Failed to ban user: %s\n", resp.Message)
		os.Exit(1)
	}
}

func unbanUser(args []string) {
	fs := flag.NewFlagSet("unban-user", flag.ExitOnError)
	requesterID := fs.String("requester-id", "", "Admin user ID")
	userID := fs.String("id", "", "User ID to unban")
	fs.Parse(args)

	if *requesterID == "" || *userID == "" {
		fmt.Fprintln(os.Stderr, "Error: --requester-id and --id are required")
		os.Exit(1)
	}

	conn, client, err := connectUserService()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	ctx, cancel := context.WithTimeout(context.Background(), getTimeout())
	defer cancel()

	resp, err := client.AdminUnbanUser(ctx, &pb.AdminUnbanUserRequest{
		RequesterUserId: *requesterID,
		UserId:          *userID,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", status.Convert(err).Message())
		os.Exit(1)
	}

	if resp.Success {
		fmt.Printf("User %s unbanned successfully\n", *userID)
	} else {
		fmt.Fprintf(os.Stderr, "Failed to unban user: %s\n", resp.Message)
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
