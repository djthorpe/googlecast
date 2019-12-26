/*
	Go Language Raspberry Pi Interface
	(c) Copyright David Thorpe 2019
	All Rights Reserved
	Documentation http://djthorpe.github.io/gopi/
	For Licensing and Usage information, please see LICENSE.md
*/

package googlecast

import (
	"context"
	"fmt"
	"time"

	// Frameworks
	googlecast "github.com/djthorpe/googlecast"
	gopi "github.com/djthorpe/gopi"
	grpc "github.com/djthorpe/gopi-rpc/sys/grpc"
	event "github.com/djthorpe/gopi/util/event"

	// Protocol buffers
	pb "github.com/djthorpe/googlecast/rpc/protobuf/googlecast"
	empty "github.com/golang/protobuf/ptypes/empty"
)

////////////////////////////////////////////////////////////////////////////////
// TYPES

type Client struct {
	pb.GoogleCastClient
	gopi.RPCClientConn
	event.Publisher
}

////////////////////////////////////////////////////////////////////////////////
// NEW

func NewGoogleCastClient(conn gopi.RPCClientConn) gopi.RPCClient {
	return &Client{
		GoogleCastClient: pb.NewGoogleCastClient(conn.(grpc.GRPCClientConn).GRPCConn()),
		RPCClientConn:    conn,
	}
}

func (this *Client) NewContext(timeout time.Duration) context.Context {
	if timeout == 0 {
		timeout = this.RPCClientConn.Timeout()
	}
	if timeout == 0 {
		return context.Background()
	} else {
		ctx, _ := context.WithTimeout(context.Background(), timeout)
		return ctx
	}
}

////////////////////////////////////////////////////////////////////////////////
// PROPERTIES

func (this *Client) Conn() gopi.RPCClientConn {
	return this.RPCClientConn
}

////////////////////////////////////////////////////////////////////////////////
// CALLS

func (this *Client) Ping() error {
	this.RPCClientConn.Lock()
	defer this.RPCClientConn.Unlock()

	// Perform ping
	if _, err := this.GoogleCastClient.Ping(this.NewContext(0), &empty.Empty{}); err != nil {
		return err
	} else {
		return nil
	}
}

func (this *Client) Devices() ([]googlecast.Device, error) {
	this.RPCClientConn.Lock()
	defer this.RPCClientConn.Unlock()

	// Get devices
	if devices, err := this.GoogleCastClient.Devices(this.NewContext(0), &empty.Empty{}); err != nil {
		return nil, err
	} else {
		return fromProtoDevicesReply(devices), nil
	}
}

////////////////////////////////////////////////////////////////////////////////
// STRINGIFY

func (this *Client) String() string {
	return fmt.Sprintf("<rpc.service.googlecast.Client>{ conn=%v }", this.RPCClientConn)
}
