package contract

import (
	"context"
	"net"
	"net/http"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	biometricpb "github.com/MAMUER/project/api/gen/biometric"
	trainingpb "github.com/MAMUER/project/api/gen/training"
	userpb "github.com/MAMUER/project/api/gen/user"
)

type contractMethod struct {
	name string
	fn   func(ctx context.Context) (interface{}, error)
}

func nilRequestContract[Req any, Resp any](
	name string,
	call func(ctx context.Context, req *Req, opts ...grpc.CallOption) (*Resp, error),
) contractMethod {
	return contractMethod{
		name: name,
		fn: func(ctx context.Context) (interface{}, error) {
			return call(ctx, nil)
		},
	}
}

func dialContractService(t *testing.T, addr, name string) *grpc.ClientConn {
	t.Helper()
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("%s not reachable: %v", name, err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

// ensureServiceReachable skips the test when the service is not running.
// Without this check a transport error ("connection refused") is indistinguishable
// from a server-side validation error and nil-request assertions pass vacuously.
func ensureServiceReachable(t *testing.T, addr, name string) {
	t.Helper()
	conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
	if err != nil {
		t.Skipf("%s not reachable at %s (%v): contract assertions require a running service", name, addr, err)
		return
	}
	_ = conn.Close()
}

func runNilRequestContract(t *testing.T, methods []contractMethod) {
	t.Helper()
	for _, m := range methods {
		t.Run(m.name, func(t *testing.T) {
			_, err := m.fn(context.Background())
			if err == nil {
				t.Errorf("%s: expected error with nil request, got nil", m.name)
			}
			t.Logf("%s contract validation: %v", m.name, err)
		})
	}
}

func TestUserServiceContract(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping contract test in short mode")
	}

	ensureServiceReachable(t, "localhost:50051", "user-service")

	client := userpb.NewUserServiceClient(dialContractService(t, "localhost:50051", "user-service"))

	runNilRequestContract(t, []contractMethod{
		nilRequestContract("Register", client.Register),
		nilRequestContract("Login", client.Login),
		nilRequestContract("GetProfile", client.GetProfile),
		nilRequestContract("UpdateProfile", client.UpdateProfile),
		nilRequestContract("ListDevices", client.ListDevices),
	})
}

func TestBiometricServiceContract(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping contract test in short mode")
	}

	ensureServiceReachable(t, "localhost:50052", "biometric-service")

	client := biometricpb.NewBiometricServiceClient(dialContractService(t, "localhost:50052", "biometric-service"))

	runNilRequestContract(t, []contractMethod{
		nilRequestContract("AddRecord", client.AddRecord),
		nilRequestContract("GetRecords", client.GetRecords),
		nilRequestContract("GetLatest", client.GetLatest),
	})
}

func TestTrainingServiceContract(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping contract test in short mode")
	}

	ensureServiceReachable(t, "localhost:50053", "training-service")

	client := trainingpb.NewTrainingServiceClient(dialContractService(t, "localhost:50053", "training-service"))

	runNilRequestContract(t, []contractMethod{
		nilRequestContract("GeneratePlan", client.GeneratePlan),
		nilRequestContract("GetPlan", client.GetPlan),
		nilRequestContract("ListPlans", client.ListPlans),
		nilRequestContract("CompleteWorkout", client.CompleteWorkout),
		nilRequestContract("GetProgress", client.GetProgress),
	})
}

func TestClassifierContract(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping contract test in short mode")
	}

	resp, err := httpGet("http://localhost:8001/health")
	if err != nil {
		t.Skipf("classifier not reachable, skipping contract test: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

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
