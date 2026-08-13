package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/MAMUER/project/internal/auth/claims"
	"github.com/MAMUER/project/internal/auth/jwt"
	"github.com/MAMUER/project/internal/cache"
	"github.com/MAMUER/project/internal/logger"

	biometricpb "github.com/MAMUER/project/api/gen/biometric"
	trainingpb "github.com/MAMUER/project/api/gen/training"
)

// ========== Mock Redis Command ==========

type mockRedisCmd struct {
	resultVal interface{}
	errVal    error
}

func (m *mockRedisCmd) Result() (interface{}, error) {
	return m.resultVal, m.errVal
}

func (m *mockRedisCmd) Err() error {
	return m.errVal
}

func (m *mockRedisCmd) String() string {
	return fmt.Sprintf("%v", m.resultVal)
}

func (m *mockRedisCmd) Val() string {
	if s, ok := m.resultVal.(string); ok {
		return s
	}
	return ""
}

// ========== Mock Redis Pipeline ==========

type mockRedisPipeline struct {
	cmds []func() *mockRedisCmd
}

func (p *mockRedisPipeline) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *mockRedisCmd {
	cmd := &mockRedisCmd{resultVal: "OK"}
	p.cmds = append(p.cmds, func() *mockRedisCmd { return cmd })
	return cmd
}

func (p *mockRedisPipeline) Get(ctx context.Context, key string) *mockRedisCmd {
	cmd := &mockRedisCmd{resultVal: "", errVal: fmt.Errorf("redis: nil")}
	p.cmds = append(p.cmds, func() *mockRedisCmd { return cmd })
	return cmd
}

func (p *mockRedisPipeline) Del(ctx context.Context, keys ...string) *mockRedisCmd {
	cmd := &mockRedisCmd{resultVal: int64(0)}
	p.cmds = append(p.cmds, func() *mockRedisCmd { return cmd })
	return cmd
}

func (p *mockRedisPipeline) Expire(ctx context.Context, key string, expiration time.Duration) *mockRedisCmd {
	cmd := &mockRedisCmd{resultVal: true}
	p.cmds = append(p.cmds, func() *mockRedisCmd { return cmd })
	return cmd
}

func (p *mockRedisPipeline) SAdd(ctx context.Context, key string, members ...interface{}) *mockRedisCmd {
	cmd := &mockRedisCmd{resultVal: int64(0)}
	p.cmds = append(p.cmds, func() *mockRedisCmd { return cmd })
	return cmd
}

func (p *mockRedisPipeline) SIsMember(ctx context.Context, key string, member interface{}) *mockRedisCmd {
	cmd := &mockRedisCmd{resultVal: false}
	p.cmds = append(p.cmds, func() *mockRedisCmd { return cmd })
	return cmd
}

func (p *mockRedisPipeline) SRem(ctx context.Context, key string, members ...interface{}) *mockRedisCmd {
	cmd := &mockRedisCmd{resultVal: int64(1)}
	p.cmds = append(p.cmds, func() *mockRedisCmd { return cmd })
	return cmd
}

func (p *mockRedisPipeline) Exec(ctx context.Context) ([]interface{}, error) {
	results := make([]interface{}, len(p.cmds))
	for i, cmdFn := range p.cmds {
		results[i] = cmdFn()
	}
	return results, nil
}

// ========== Mock Redis Client ==========

type mockRedisClient struct {
	data   map[string]string
	sets   map[string]map[string]bool
	incr   map[string]int64
	err    error
}

func (m *mockRedisClient) Pipeline() *mockRedisPipeline {
	return &mockRedisPipeline{}
}

func (m *mockRedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *mockRedisCmd {
	if m.err != nil {
		return &mockRedisCmd{errVal: m.err}
	}
	m.data[key] = fmt.Sprintf("%v", value)
	return &mockRedisCmd{resultVal: "OK"}
}

func (m *mockRedisClient) Get(ctx context.Context, key string) *mockRedisCmd {
	if m.err != nil {
		return &mockRedisCmd{errVal: m.err}
	}
	val, ok := m.data[key]
	if !ok {
		return &mockRedisCmd{errVal: fmt.Errorf("redis: nil")}
	}
	return &mockRedisCmd{resultVal: val}
}

func (m *mockRedisClient) Del(ctx context.Context, keys ...string) *mockRedisCmd {
	if m.err != nil {
		return &mockRedisCmd{errVal: m.err}
	}
	count := int64(0)
	for _, key := range keys {
		if _, ok := m.data[key]; ok {
			delete(m.data, key)
			count++
		}
	}
	return &mockRedisCmd{resultVal: count}
}

func (m *mockRedisClient) Expire(ctx context.Context, key string, expiration time.Duration) *mockRedisCmd {
	if m.err != nil {
		return &mockRedisCmd{errVal: m.err}
	}
	return &mockRedisCmd{resultVal: true}
}

func (m *mockRedisClient) SAdd(ctx context.Context, key string, members ...interface{}) *mockRedisCmd {
	if m.err != nil {
		return &mockRedisCmd{errVal: m.err}
	}
	if _, ok := m.sets[key]; !ok {
		m.sets[key] = make(map[string]bool)
	}
	count := int64(0)
	for _, member := range members {
		s := fmt.Sprintf("%v", member)
		if !m.sets[key][s] {
			m.sets[key][s] = true
			count++
		}
	}
	return &mockRedisCmd{resultVal: count}
}

func (m *mockRedisClient) SIsMember(ctx context.Context, key string, member interface{}) *mockRedisCmd {
	if m.err != nil {
		return &mockRedisCmd{errVal: m.err}
	}
	s := fmt.Sprintf("%v", member)
	if set, ok := m.sets[key]; ok {
		return &mockRedisCmd{resultVal: set[s]}
	}
	return &mockRedisCmd{resultVal: false}
}

func (m *mockRedisClient) SRem(ctx context.Context, key string, members ...interface{}) *mockRedisCmd {
	if m.err != nil {
		return &mockRedisCmd{errVal: m.err}
	}
	count := int64(0)
	for _, member := range members {
		s := fmt.Sprintf("%v", member)
		if set, ok := m.sets[key]; ok && set[s] {
			delete(set, s)
			count++
		}
	}
	return &mockRedisCmd{resultVal: count}
}

func (m *mockRedisClient) Scan(ctx context.Context, cursor uint64, match string, count int64) ([]string, uint64, error) {
	if m.err != nil {
		return nil, 0, m.err
	}
	var keys []string
	for key := range m.data {
		keys = append(keys, key)
	}
	return keys, 0, nil
}

func (m *mockRedisClient) Incr(ctx context.Context, key string) *mockRedisCmd {
	if m.err != nil {
		return &mockRedisCmd{errVal: m.err}
	}
	m.incr[key]++
	return &mockRedisCmd{resultVal: m.incr[key]}
}

// ========== Mock Session Store ==========

type mockSessionStore struct {
	sessions        map[string]string
	criticalSessions map[string]string
	validateErr     error
	createErr       error
	invalidateErr   error
}

func (m *mockSessionStore) InvalidateUserSession(ctx context.Context, userID string) error {
	if m.invalidateErr != nil {
		return m.invalidateErr
	}
	delete(m.sessions, userID)
	return nil
}

func (m *mockSessionStore) ValidateCriticalSession(ctx context.Context, token, expectedUserID string) error {
	if m.validateErr != nil {
		return m.validateErr
	}
	userID, ok := m.criticalSessions[token]
	if !ok {
		return errors.New("session expired")
	}
	if userID != expectedUserID {
		return errors.New("invalid session")
	}
	delete(m.criticalSessions, token)
	return nil
}

func (m *mockSessionStore) CreateCriticalSession(ctx context.Context, userID string) (string, error) {
	if m.createErr != nil {
		return "", m.createErr
	}
	token := "critical-token-" + userID
	m.criticalSessions[token] = userID
	return token, nil
}

// ========== Mock Token Provider ==========

type mockTokenProvider struct{}

func (m *mockTokenProvider) GenerateAccessToken(userID, email, role string, ttl time.Duration) (string, error) {
	return "mock-access-token-" + userID, nil
}

func (m *mockTokenProvider) GenerateRefreshToken() string {
	return "mock-refresh-token"
}

func (m *mockTokenProvider) ValidateAccessToken(token string) (*claims.Claims, error) {
	return nil, nil
}

func (m *mockTokenProvider) ComputeTokenFingerprint(token string) string {
	return "mock-fingerprint-" + token
}

func (m *mockTokenProvider) PublicKeyPEMToJWKS(publicKeyPEM string) ([]byte, error) {
	return nil, nil
}

func (m *mockTokenProvider) PublicKeyPEM() string {
	return "mock-public-key"
}

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
		Id:        "rec-" + req.UserId,
		UserId:    req.UserId,
		MetricType: req.MetricType,
		Value:     req.Value,
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
		Id: req.Id,
		Value: req.Value,
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
	plans     map[string]*trainingpb.TrainingPlan
	planData  *trainingpb.GeneratePlanResponse
	listResp  *trainingpb.ListPlansResponse
	progress  *trainingpb.GetProgressResponse
	completeErr error
	getErr   error
	listErr  error
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

// ========== Mock Reverse Proxy ==========

type mockReverseProxy struct {
	handler http.Handler
}

func (m *mockReverseProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if m.handler != nil {
		m.handler.ServeHTTP(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

// ========== Test Gateway Setup ==========

func newTestGateway(opts ...func(*gateway)) *gateway {
	log := &logger.Logger{Logger: zap.NewNop()}
	privateKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	privateKeyBytes, _ := x509.MarshalECPrivateKey(privateKey)
	privateKeyPEM := string(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: privateKeyBytes}))
	publicKeyBytes, _ := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	publicKeyPEM := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicKeyBytes}))

	g := &gateway{
		log:           log,
		userClient:    &mockUserServiceClient{},
		tokenProvider: jwt.NewJWTAdapter(privateKeyPEM, publicKeyPEM),
	}
	for _, opt := range opts {
		opt(g)
	}
	return g
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
