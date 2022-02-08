package proxy

import (
	"context"
	"net"
	"testing"

	"github.com/gravitational/teleport/api/client/proto"
	"github.com/gravitational/teleport/api/types"
	"github.com/gravitational/teleport/lib/reversetunnel"
	"github.com/gravitational/trace"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type mockClusterDialer struct {
	MockDialCluster clusterDialerFunc
}

func (m *mockClusterDialer) Dial(clusterName string, request reversetunnel.DialParams) (net.Conn, error) {
	if m.MockDialCluster == nil {
		return nil, trace.NotImplemented("")
	}
	return m.MockDialCluster(clusterName, request)
}

func setupService(t *testing.T) (*proxyService, proto.ProxyServiceClient) {
	server := grpc.NewServer()
	t.Cleanup(server.Stop)

	listener, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)

	proxyService := &proxyService{
		log: logrus.New(),
	}
	proto.RegisterProxyServiceServer(server, proxyService)

	go server.Serve(listener)

	conn, err := grpc.Dial(listener.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })

	client := proto.NewProxyServiceClient(conn)
	return proxyService, client
}

func TestInvalidFirstFrame(t *testing.T) {
	_, client := setupService(t)
	stream, err := client.DialNode(context.TODO())
	require.NoError(t, err)

	err = stream.Send(&proto.Frame{
		Message: &proto.Frame_Data{},
	})
	require.NoError(t, err)

	_, err = stream.Recv()
	require.Error(t, err, "expected invalid dial request")
}

func TestSendReceive(t *testing.T) {
	service, client := setupService(t)
	stream, err := client.DialNode(context.TODO())
	require.NoError(t, err)

	dialRequest := &proto.DialRequest{
		NodeID:      "test-id.test-cluster",
		TunnelType:  types.NodeTunnel,
		Source:      &proto.NetAddr{},
		Destination: &proto.NetAddr{},
	}

	local, remote := net.Pipe()
	service.clusterDialer = &mockClusterDialer{
		MockDialCluster: func(clusterName string, request reversetunnel.DialParams) (net.Conn, error) {
			require.Equal(t, "test-cluster", clusterName)
			require.Equal(t, dialRequest.TunnelType, request.ConnType)
			require.Equal(t, dialRequest.NodeID, request.ServerID)

			return remote, nil
		},
	}

	send := []byte("ping")
	recv := []byte("pong")

	err = stream.Send(&proto.Frame{Message: &proto.Frame_DialRequest{
		DialRequest: dialRequest,
	}})
	require.NoError(t, err)

	err = stream.Send(&proto.Frame{Message: &proto.Frame_Data{Data: &proto.Data{
		Bytes: send,
	}}})
	require.NoError(t, err)

	b := make([]byte, 4)
	local.Read(b)
	require.Equal(t, send, b, "unexpected bytes sent")

	local.Write(recv)
	msg, err := stream.Recv()
	require.NoError(t, err)
	require.Equal(t, recv, msg.GetData().Bytes, "unexpected bytes received")
}
