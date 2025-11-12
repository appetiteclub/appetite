package authn

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	authnpb "github.com/appetiteclub/appetite/services/authn/internal/authn/proto"
	"github.com/aquamarinepk/aqm"
	authpkg "github.com/aquamarinepk/aqm/auth"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// GRPCServer exposes the AuthN feature over gRPC and integrates with the service lifecycle.
type GRPCServer struct {
	authnpb.UnimplementedAuthServer
	authnpb.UnimplementedUsersServer
	authnpb.UnimplementedSystemServer

	repo   UserRepo
	logger aqm.Logger
	config  *aqm.Config

	mu  sync.Mutex
	srv *grpc.Server
	lis net.Listener
}

func NewGRPCServer(repo UserRepo, logger aqm.Logger, config *aqm.Config) *GRPCServer {
	if logger == nil {
		logger = aqm.NewNoopLogger()
	}
	return &GRPCServer{
		repo:   repo,
		logger: logger,
		config:  config,
	}
}

// Start launches the gRPC server and registers all feature services.
func (s *GRPCServer) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.srv != nil {
		return errors.New("gRPC server already started")
	}

	grpcPort, _ := s.config.GetString("grpc.port")
	addr := grpcPort
	if strings.TrimSpace(addr) == "" {
		addr = ":9082"
	}

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen gRPC on %s: %w", addr, err)
	}

	server := grpc.NewServer()

	authnpb.RegisterAuthServer(server, s)
	authnpb.RegisterUsersServer(server, s)
	authnpb.RegisterSystemServer(server, s)

	logger := s.logger.With("transport", "grpc", "address", addr)

	s.srv = server
	s.lis = lis

	go func() {
		logger.Info("gRPC server listening")
		if err := server.Serve(lis); err != nil {
			if !errors.Is(err, grpc.ErrServerStopped) {
				logger.Error("gRPC server stopped with error", "error", err)
			}
		}
	}()

	go func() {
		<-ctx.Done()
		s.Stop(context.Background())
	}()

	return nil
}

// Stop gracefully stops the gRPC server.
func (s *GRPCServer) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.srv == nil {
		return nil
	}

	done := make(chan struct{})
	go func(server *grpc.Server) {
		server.GracefulStop()
		close(done)
	}(s.srv)

	select {
	case <-done:
	case <-ctx.Done():
		s.srv.Stop()
	case <-time.After(5 * time.Second):
		s.srv.Stop()
	}

	if s.lis != nil {
		_ = s.lis.Close()
	}

	s.srv = nil
	s.lis = nil

	return nil
}

func (s *GRPCServer) SignUp(ctx context.Context, req *authnpb.SignUpRequest) (*authnpb.AuthResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request cannot be nil")
	}

	username := deriveUsernameFromEmail(req.Email)
	name := fallbackNameFromEmail(req.Email)
	if errs := ValidateSignUpRequest(req.Email, req.Password, username, name); len(errs) > 0 {
		return nil, status.Errorf(codes.InvalidArgument, "validation failed: %s", joinValidationMessages(errs))
	}

	user, err := SignUpUser(ctx, s.repo, s.config, req.Email, req.Password, username, name)
	if err != nil {
		switch {
		case errors.Is(err, ErrUserExists):
			return nil, status.Error(codes.AlreadyExists, "user already exists")
		case errors.Is(err, ErrUsernameExists):
			return nil, status.Error(codes.AlreadyExists, "username already exists")
		default:
			return nil, status.Errorf(codes.Internal, "could not create account: %v", err)
		}
	}

	return &authnpb.AuthResponse{User: toProtoUser(user)}, nil
}

func (s *GRPCServer) SignIn(ctx context.Context, req *authnpb.SignInRequest) (*authnpb.AuthResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request cannot be nil")
	}

	if errs := ValidateSignInRequest(req.Email, req.Password); len(errs) > 0 {
		return nil, status.Errorf(codes.InvalidArgument, "validation failed: %s", joinValidationMessages(errs))
	}

	user, token, err := SignInUser(ctx, s.repo, s.config, req.Email, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidCredentials):
			return nil, status.Error(codes.Unauthenticated, "invalid credentials")
		case errors.Is(err, ErrInactiveAccount):
			return nil, status.Error(codes.PermissionDenied, "account is not active")
		default:
			return nil, status.Errorf(codes.Internal, "authentication failed: %v", err)
		}
	}

	return &authnpb.AuthResponse{User: toProtoUser(user), Token: token}, nil
}

func (s *GRPCServer) SignOut(context.Context, *emptypb.Empty) (*emptypb.Empty, error) {
	// TODO: Implement token invalidation when session management is available.
	return &emptypb.Empty{}, nil
}

func (s *GRPCServer) Create(ctx context.Context, req *authnpb.CreateUserRequest) (*authnpb.User, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request cannot be nil")
	}

	validation := ValidateCreateUserRequest(ctx, UserCreateRequest{
		Name:     req.Name,
		Email:    req.Email,
		Password: req.Password,
		Status:   req.Status,
		Username: deriveUsernameFromEmail(req.Email),
	})
	if len(validation) > 0 {
		return nil, status.Errorf(codes.InvalidArgument, "validation failed: %s", strings.Join(validation, ", "))
	}

	user, err := SignUpUser(ctx, s.repo, s.config, req.Email, req.Password, deriveUsernameFromEmail(req.Email), req.Name)
	if err != nil {
		switch {
		case errors.Is(err, ErrUserExists):
			return nil, status.Error(codes.AlreadyExists, "user already exists")
		case errors.Is(err, ErrUsernameExists):
			return nil, status.Error(codes.AlreadyExists, "username already exists")
		default:
			return nil, status.Errorf(codes.Internal, "could not create user: %v", err)
		}
	}

	if strings.TrimSpace(req.Status) != "" && string(user.Status) != req.Status {
		user.Status = authpkg.UserStatus(req.Status)
		if saveErr := s.repo.Save(ctx, user); saveErr != nil {
			return nil, status.Errorf(codes.Internal, "could not update user status: %v", saveErr)
		}
	}

	return toProtoUser(user), nil
}

func (s *GRPCServer) Get(ctx context.Context, req *authnpb.GetUserRequest) (*authnpb.User, error) {
	if req == nil || strings.TrimSpace(req.Id) == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid id format")
	}

	user, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not retrieve user: %v", err)
	}
	if user == nil {
		return nil, status.Error(codes.NotFound, "user not found")
	}

	return toProtoUser(user), nil
}

func (s *GRPCServer) List(ctx context.Context, req *authnpb.ListUsersRequest) (*authnpb.ListUsersResponse, error) {
	var (
		users []*User
		err   error
	)

	if req != nil && strings.TrimSpace(req.Status) != "" {
		users, err = s.repo.ListByStatus(ctx, req.Status)
	} else {
		users, err = s.repo.List(ctx)
	}

	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not list users: %v", err)
	}

	resp := &authnpb.ListUsersResponse{}
	for _, user := range users {
		resp.Users = append(resp.Users, toProtoUser(user))
	}

	return resp, nil
}

func (s *GRPCServer) Update(ctx context.Context, req *authnpb.UpdateUserRequest) (*authnpb.User, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request cannot be nil")
	}

	if strings.TrimSpace(req.Id) == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid id format")
	}

	if strings.TrimSpace(req.Email) != "" {
		return nil, status.Error(codes.InvalidArgument, "email updates are not supported yet")
	}

	if strings.TrimSpace(req.Password) != "" {
		if errs := authpkg.ValidatePassword(req.Password); len(errs) > 0 {
			return nil, status.Errorf(codes.InvalidArgument, "invalid password: %s", joinAuthErrors(errs))
		}
	}

	var newName string
	if trimmed := strings.TrimSpace(req.Name); trimmed != "" {
		if len([]rune(trimmed)) < 2 {
			return nil, status.Error(codes.InvalidArgument, "name must be at least 2 characters")
		}
		if len([]rune(trimmed)) > 120 {
			return nil, status.Error(codes.InvalidArgument, "name must be 120 characters or less")
		}
		newName = trimmed
	}

	if strings.TrimSpace(req.Status) != "" {
		if errs := authpkg.ValidateUserStatus(authpkg.UserStatus(req.Status)); len(errs) > 0 {
			return nil, status.Error(codes.InvalidArgument, "invalid status")
		}
	}

	user, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "could not load user: %v", err)
	}
	if user == nil {
		return nil, status.Error(codes.NotFound, "user not found")
	}

	if strings.TrimSpace(req.Status) != "" {
		user.Status = authpkg.UserStatus(req.Status)
	}

	if newName != "" {
		user.Name = newName
	}

	if strings.TrimSpace(req.Password) != "" {
		salt := authpkg.GeneratePasswordSalt()
		user.PasswordSalt = salt
		user.PasswordHash = authpkg.HashPassword([]byte(req.Password), salt)
	}

	if err := s.repo.Save(ctx, user); err != nil {
		return nil, status.Errorf(codes.Internal, "could not update user: %v", err)
	}

	return toProtoUser(user), nil
}

func (s *GRPCServer) Delete(ctx context.Context, req *authnpb.DeleteUserRequest) (*emptypb.Empty, error) {
	if req == nil || strings.TrimSpace(req.Id) == "" {
		return nil, status.Error(codes.InvalidArgument, "id is required")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid id format")
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return nil, status.Errorf(codes.Internal, "could not delete user: %v", err)
	}

	return &emptypb.Empty{}, nil
}

func (s *GRPCServer) GetBootstrapStatus(ctx context.Context, _ *emptypb.Empty) (*authnpb.BootstrapStatusResponse, error) {
	user, err := GenerateBootstrapStatus(ctx, s.repo, s.config)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to compute bootstrap status: %v", err)
	}

	if user == nil {
		return &authnpb.BootstrapStatusResponse{NeedsBootstrap: true}, nil
	}

	return &authnpb.BootstrapStatusResponse{
		NeedsBootstrap: false,
		SuperadminId:   user.ID.String(),
	}, nil
}

func (s *GRPCServer) Bootstrap(ctx context.Context, _ *emptypb.Empty) (*authnpb.BootstrapResponse, error) {
	user, password, err := BootstrapSuperadmin(ctx, s.repo, s.config)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to bootstrap superadmin: %v", err)
	}

	return &authnpb.BootstrapResponse{
		SuperadminId: user.ID.String(),
		Email:        SuperadminEmail,
		Password:     password,
	}, nil
}

func toProtoUser(user *User) *authnpb.User {
	if user == nil {
		return nil
	}

	protoUser := &authnpb.User{
		Id:        user.ID.String(),
		Status:    string(user.Status),
		CreatedBy: user.CreatedBy,
		UpdatedBy: user.UpdatedBy,
	}

	if !user.CreatedAt.IsZero() {
		protoUser.CreatedAt = timestamppb.New(user.CreatedAt)
	}
	if !user.UpdatedAt.IsZero() {
		protoUser.UpdatedAt = timestamppb.New(user.UpdatedAt)
	}

	return protoUser
}

func joinValidationMessages(errs []ValidationError) string {
	messages := make([]string, 0, len(errs))
	for _, err := range errs {
		if strings.TrimSpace(err.Message) != "" {
			if strings.TrimSpace(err.Field) != "" {
				messages = append(messages, fmt.Sprintf("%s: %s", err.Field, err.Message))
			} else {
				messages = append(messages, err.Message)
			}
		}
	}
	return strings.Join(messages, "; ")
}

func joinAuthErrors(errs []authpkg.ValidationError) string {
	messages := make([]string, 0, len(errs))
	for _, err := range errs {
		if strings.TrimSpace(err.Message) != "" {
			messages = append(messages, err.Message)
		}
	}
	return strings.Join(messages, "; ")
}
