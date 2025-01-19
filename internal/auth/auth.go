package auth

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/Jamolkhon5/mistral/pkg/proto/auth_v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/emptypb"
)

var gClient auth_v1.AuthV1Client

func InitClient(conn *grpc.ClientConn) {
	gClient = auth_v1.NewAuthV1Client(conn)
}

func VerifyToken(r *http.Request) (string, error) {
	authToken := r.Header.Get("Authorization")
	if authToken == "" {
		return "", fmt.Errorf("missing authorization header")
	}

	md := metadata.New(map[string]string{
		"Authorization": authToken,
	})
	ctx := metadata.NewOutgoingContext(context.Background(), md)

	userInfo, err := gClient.GetUser(ctx, &emptypb.Empty{})
	if err != nil {
		log.Printf("Authentication failed: %v", err)
		return "", err
	}

	return userInfo.GetUser().GetId(), nil
}
