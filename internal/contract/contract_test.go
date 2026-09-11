package contract

import (
	"context"
	"net/http"
	"testing"

	userpb "github.com/MAMUER/project/api/gen/user"
	biometricpb "github.com/MAMUER/project/api/gen/biometric"
	trainingpb "github.com/MAMUER/project/api/gen/training"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestUserServiceContract(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping contract test in short mode")
	}

	conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("failed to connect to user-service: %v", err)
	}
	defer conn.Close()

	client := userpb.NewUserServiceClient(conn)

		methods := []struct {
			name string
			fn   func(ctx context.Context, req interface{}) (interface{}, error)
		}{
			{"Register", func(ctx context.Context, req interface{}) (interface{}, error) {
				return client.Register(ctx, req.(*userpb.RegisterRequest))
			}},
			{"Login", func(ctx context.Context, req interface{}) (interface{}, error) {
				return client.Login(ctx, req.(*userpb.LoginRequest))
			}},
			{"GetProfile", func(ctx context.Context, req interface{}) (interface{}, error) {
				return client.GetProfile(ctx, req.(*userpb.GetProfileRequest))
			}},
			{"UpdateProfile", func(ctx context.Context, req interface{}) (interface{}, error) {
				return client.UpdateProfile(ctx, req.(*userpb.UpdateProfileRequest))
			}},
			{"ListDevices", func(ctx context.Context, req interface{}) (interface{}, error) {
				return client.ListDevices(ctx, req.(*userpb.ListDevicesRequest))
			}},
	}

	ctx := context.Background()
	for _, m := range methods {
		t.Run(m.name, func(t *testing.T) {
			_, err := m.fn(ctx, nil)
			if err == nil {
				t.Errorf("%s: expected error with nil request, got nil", m.name)
			}
			t.Logf("%s contract validation: %v", m.name, err)
		})
	}
}

func TestBiometricServiceContract(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping contract test in short mode")
	}

	conn, err := grpc.Dial("localhost:50052", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("failed to connect to biometric-service: %v", err)
	}
	defer conn.Close()

		client := biometricpb.NewBiometricServiceClient(conn)

		methods := []struct {
			name string
			fn   func(ctx context.Context, req interface{}) (interface{}, error)
		}{
			{"AddRecord", func(ctx context.Context, req interface{}) (interface{}, error) {
				return client.AddRecord(ctx, req.(*biometricpb.AddRecordRequest))
			}},
			{"GetRecords", func(ctx context.Context, req interface{}) (interface{}, error) {
				return client.GetRecords(ctx, req.(*biometricpb.GetRecordsRequest))
			}},
			{"GetLatest", func(ctx context.Context, req interface{}) (interface{}, error) {
				return client.GetLatest(ctx, req.(*biometricpb.GetLatestRequest))
			}},
	}

	ctx := context.Background()
	for _, m := range methods {
		t.Run(m.name, func(t *testing.T) {
			_, err := m.fn(ctx, nil)
			if err == nil {
				t.Errorf("%s: expected error with nil request, got nil", m.name)
			}
			t.Logf("%s contract validation: %v", m.name, err)
		})
	}
}

func TestTrainingServiceContract(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping contract test in short mode")
	}

	conn, err := grpc.Dial("localhost:50053", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("failed to connect to training-service: %v", err)
	}
	defer conn.Close()

		client := trainingpb.NewTrainingServiceClient(conn)

		methods := []struct {
			name string
			fn   func(ctx context.Context, req interface{}) (interface{}, error)
		}{
			{"GeneratePlan", func(ctx context.Context, req interface{}) (interface{}, error) {
				return client.GeneratePlan(ctx, req.(*trainingpb.GeneratePlanRequest))
			}},
			{"GetPlan", func(ctx context.Context, req interface{}) (interface{}, error) {
				return client.GetPlan(ctx, req.(*trainingpb.GetPlanRequest))
			}},
			{"ListPlans", func(ctx context.Context, req interface{}) (interface{}, error) {
				return client.ListPlans(ctx, req.(*trainingpb.ListPlansRequest))
			}},
			{"CompleteWorkout", func(ctx context.Context, req interface{}) (interface{}, error) {
				return client.CompleteWorkout(ctx, req.(*trainingpb.CompleteWorkoutRequest))
			}},
			{"GetProgress", func(ctx context.Context, req interface{}) (interface{}, error) {
				return client.GetProgress(ctx, req.(*trainingpb.GetProgressRequest))
			}},
	}

	ctx := context.Background()
	for _, m := range methods {
		t.Run(m.name, func(t *testing.T) {
			_, err := m.fn(ctx, nil)
			if err == nil {
				t.Errorf("%s: expected error with nil request, got nil", m.name)
			}
			t.Logf("%s contract validation: %v", m.name, err)
		})
	}
}

func TestClassifierContract(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping contract test in short mode")
	}

	resp, err := httpGet("http://localhost:8001/health")
	if err != nil {
		t.Fatalf("classifier not reachable: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("classifier health check failed: %d", resp.StatusCode)
	}
	t.Log("classifier contract validation: health endpoint OK")
}

func httpGet(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	return http.DefaultClient.Do(req)
}
