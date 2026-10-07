package server

import (
	"context"
	"github.com/IM_System/apps/operations/rpc/internal/svc"
	"github.com/IM_System/apps/operations/rpc/operations"
	"github.com/IM_System/pkg/serviceauth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"net"
	"testing"
	"time"
)

func TestObservationContractHasNoBusinessQueries(t *testing.T) {
	methods := operations.ObservationQuery_ServiceDesc.Methods
	if len(methods) != 4 {
		t.Fatalf("observation methods=%d", len(methods))
	}
	for _, m := range methods {
		if m.MethodName == "GetMessageRecord" || m.MethodName == "FindUserReference" || m.MethodName == "SearchMessages" {
			t.Fatal("business method in observation contract", m.MethodName)
		}
	}
	s := &svc.ServiceContext{}
	s.Config.DeliveryObservation.Enabled = true
	resp, err := NewObservationServer(s).GetCapabilities(context.Background(), &operations.GetCapabilitiesRequest{})
	if err != nil || resp.MessageRecord != "unsupported" || resp.MessageSearch != "unsupported" || resp.MessageTimeline != "partial" {
		t.Fatalf("%+v %v", resp, err)
	}
}
func TestBothContractsRequireOperationsCredential(t *testing.T) {
	lis := bufconn.Listen(1 << 20)
	auth := serviceauth.NewAuth(true, "operations-secret")
	gs := grpc.NewServer(grpc.UnaryInterceptor(auth.UnaryInterceptor))
	s := &svc.ServiceContext{}
	operations.RegisterObservationQueryServer(gs, NewObservationServer(s))
	operations.RegisterOperationsQueryServer(gs, NewOperationsServer(s))
	go gs.Serve(lis)
	t.Cleanup(func() { gs.Stop(); lis.Close() })
	conn, err := grpc.NewClient("passthrough:///test", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() }))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for _, method := range []string{"/operations.ObservationQuery/GetCapabilities", "/operations.OperationsQuery/GetCapabilities"} {
		var resp operations.GetCapabilitiesResponse
		if err := conn.Invoke(ctx, method, &operations.GetCapabilitiesRequest{}, &resp); status.Code(err) != codes.PermissionDenied {
			t.Fatalf("%s: %v", method, err)
		}
		if err := conn.Invoke(serviceauth.Outgoing(ctx, "operations-secret"), method, &operations.GetCapabilitiesRequest{}, &resp); err != nil {
			t.Fatalf("%s: %v", method, err)
		}
	}
}
