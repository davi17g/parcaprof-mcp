package parca

import (
	"context"
	"crypto/tls"
	"fmt"

	querypb "github.com/parca-dev/parca/gen/proto/go/parca/query/v1alpha1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

type Config struct {
	Address     string
	Insecure    bool
	BearerToken string
}

type Client struct {
	Conn  *grpc.ClientConn
	Query querypb.QueryServiceClient
}

func Dial(cfg Config) (*Client, error) {
	if cfg.Address == "" {
		return nil, fmt.Errorf("parca address is required")
	}
	var opts []grpc.DialOption
	if cfg.Insecure {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})))
	}
	if cfg.BearerToken != "" {
		opts = append(opts, grpc.WithPerRPCCredentials(bearer{token: cfg.BearerToken, insecure: cfg.Insecure}))
	}
	conn, err := grpc.NewClient(cfg.Address, opts...)
	if err != nil {
		return nil, fmt.Errorf("dial parca: %w", err)
	}
	return &Client{Conn: conn, Query: querypb.NewQueryServiceClient(conn)}, nil
}

func (c *Client) Close() error { return c.Conn.Close() }

type bearer struct {
	token    string
	insecure bool
}

func (b bearer) GetRequestMetadata(_ context.Context, _ ...string) (map[string]string, error) {
	return map[string]string{"authorization": "Bearer " + b.token}, nil
}

func (b bearer) RequireTransportSecurity() bool { return !b.insecure }
