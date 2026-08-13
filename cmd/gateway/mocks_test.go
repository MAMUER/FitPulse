package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"net/http/httputil"
	"net/url"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	biometricpb "github.com/MAMUER/project/api/gen/biometric"
	trainingpb "github.com/MAMUER/project/api/gen/training"
	"github.com/MAMUER/project/internal/auth/jwt"
	"github.com/MAMUER/project/internal/cache"
	"github.com/MAMUER/project/internal/logger"
)

// ========== Mock Biometric Client ==========

type mockBiometricClient struct {
	records map[string]*biometricpb.BiometricRecord
	latest  *biometricpb.BiometricRecord
	addErr  error
	getErr  error
}

func newMockBiometricClient() *mockBiometricClient {
	return &mockBiometricClient{
		records: make(map[string]*biometricpb.BiometricRecord),
	}
}

func (m *mockBiometricClient) AddRecord(ctx context.Context, req *biometricpb.AddRecordRequest, opts ...grpc.CallOption) (*biometricpb.AddRecordResponse, error) {
	if m.addErr != nil {
		return nil, m.addErr
	}
	record := &biometricpb.BiometricRecord{
		Id:         "rec-" + req.UserId,
		UserId:     req.UserId,
		MetricType: req.MetricType,
		Value:      req.Value,
		DeviceType: req.DeviceType,
	}
	m.records[req.MetricType] = record
	return &biometricpb.AddRecordResponse{Id: record.Id}, nil
}

func (m *mockBiometricClient) GetLatest(ctx context.Context, req *biometricpb.GetLatestRequest, opts ...grpc.CallOption) (*biometricpb.BiometricRecord, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	if rec, ok := m.records[req.MetricType]; ok {
		return rec, nil
	}
	if m.latest != nil {
		return m.latest, nil
	}
	return nil, status.Errorf(codes.NotFound, "not found")
}

func (m *mockBiometricClient) GetRecords(ctx context.Context, req *biometricpb.GetRecordsRequest, opts ...grpc.CallOption) (*biometricpb.GetRecordsResponse, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	var records []*biometricpb.BiometricRecord
	for _, rec := range m.records {
		if req.MetricType == "" || rec.MetricType == req.MetricType {
			records = append(records, rec)
		}
	}
	return &biometricpb.GetRecordsResponse{Records: records}, nil
}

func (m *mockBiometricClient) UpdateRecord(ctx context.Context, req *biometricpb.UpdateRecordRequest, opts ...grpc.CallOption) (*biometricpb.BiometricRecord, error) {
	if m.addErr != nil {
		return nil, m.addErr
	}
	return &biometricpb.BiometricRecord{
		Id:         req.Id,
		Value:      req.Value,
		DeviceType: req.DeviceType,
	}, nil
}

func (m *mockBiometricClient) BatchAddRecords(ctx context.Context, req *biometricpb.BatchAddRecordsRequest, opts ...grpc.CallOption) (*biometricpb.BatchAddRecordsResponse, error) {
	return nil, nil
}

func (m *mockBiometricClient) DeleteRecord(ctx context.Context, req *biometricpb.DeleteRecordRequest, opts ...grpc.CallOption) (*biometricpb.DeleteRecordResponse, error) {
	return &biometricpb.DeleteRecordResponse{Deleted: true}, nil
}

// ========== Mock Training Client ==========

type mockTrainingClient struct {
	plans       map[string]*trainingpb.TrainingPlan
	planData    *trainingpb.GeneratePlanResponse
	listResp    *trainingpb.ListPlansResponse
	progress    *trainingpb.GetProgressResponse
	completeErr error
	getErr      error
	listErr     error
	generateErr error
}

func newMockTrainingClient() *mockTrainingClient {
	return &mockTrainingClient{
		plans: make(map[string]*trainingpb.TrainingPlan),
	}
}

func (m *mockTrainingClient) GeneratePlan(ctx context.Context, req *trainingpb.GeneratePlanRequest, opts ...grpc.CallOption) (*trainingpb.GeneratePlanResponse, error) {
	if m.generateErr != nil {
		return nil, m.generateErr
	}
	if m.planData != nil {
		return m.planData, nil
	}
	return &trainingpb.GeneratePlanResponse{
		PlanId: "plan-" + req.UserId,
	}, nil
}

func (m *mockTrainingClient) GetPlan(ctx context.Context, req *trainingpb.GetPlanRequest, opts ...grpc.CallOption) (*trainingpb.TrainingPlan, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	if plan, ok := m.plans[req.PlanId]; ok {
		return plan, nil
	}
	return nil, status.Errorf(codes.NotFound, "not found")
}

func (m *mockTrainingClient) ListPlans(ctx context.Context, req *trainingpb.ListPlansRequest, opts ...grpc.CallOption) (*trainingpb.ListPlansResponse, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	if m.listResp != nil {
		return m.listResp, nil
	}
	return &trainingpb.ListPlansResponse{Plans: []*trainingpb.TrainingPlan{}, Total: 0}, nil
}

func (m *mockTrainingClient) CompleteWorkout(ctx context.Context, req *trainingpb.CompleteWorkoutRequest, opts ...grpc.CallOption) (*trainingpb.CompleteWorkoutResponse, error) {
	if m.completeErr != nil {
		return nil, m.completeErr
	}
	return &trainingpb.CompleteWorkoutResponse{Success: true}, nil
}

func (m *mockTrainingClient) GetProgress(ctx context.Context, req *trainingpb.GetProgressRequest, opts ...grpc.CallOption) (*trainingpb.GetProgressResponse, error) {
	if m.progress != nil {
		return m.progress, nil
	}
	return &trainingpb.GetProgressResponse{}, nil
}

// ========== Test Gateway Setup ==========

func newTestGateway() *gateway {
	log := &logger.Logger{Logger: zap.NewNop()}
	privateKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	privateKeyBytes, _ := x509.MarshalECPrivateKey(privateKey)
	privateKeyPEM := string(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: privateKeyBytes}))
	publicKeyBytes, _ := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	publicKeyPEM := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicKeyBytes}))

	return &gateway{
		log:           log,
		userClient:    &mockUserServiceClient{},
		tokenProvider: jwt.NewJWTAdapter(privateKeyPEM, publicKeyPEM),
	}
}

func withRealRedis(g *gateway) {
	g.valkeyDB = redis.NewClient(&redis.Options{
		Addr: "localhost:99999",
	})
}

func withRealSessionStore(g *gateway) {
	g.sessionStore = cache.NewSessionStoreFromRedis(redis.NewClient(&redis.Options{
		Addr: "localhost:99999",
	}))
}

func withBiometricClient(g *gateway) {
	g.biometricClient = newMockBiometricClient()
}

func withTrainingClient(g *gateway) {
	g.trainingClient = newMockTrainingClient()
}

func withOAuth(g *gateway) {
	conf := &oauth2.Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-secret",
		RedirectURL:  "http://localhost/callback",
		Scopes:       []string{"openid", "profile", "email"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "http://localhost/auth",
			TokenURL: "http://localhost/token",
		},
	}
	g.googleOAuthConfig = conf
}

func withProxy(g *gateway, targetURL string) {
	u, _ := url.Parse(targetURL)
	g.biometricWebhookProxy = httputil.NewSingleHostReverseProxy(u)
}

func withClassifierURL(g *gateway, url string) {
	g.classifierURL = url
}

func withMLGeneratorURL(g *gateway, url string) {
	g.mlGeneratorURL = url
}

func grpcError(code codes.Code, msg string) error {
	st := status.New(code, msg)
	return st.Err()
}
