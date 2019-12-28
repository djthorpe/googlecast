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
	"io"
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
// CONSTANTS

const (
	HEARTBEAT_TIMEOUT = 5 * time.Second
)

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

func (this *Client) StreamEvents(ctx context.Context) error {
	this.RPCClientConn.Lock()
	defer this.RPCClientConn.Unlock()

	// Errors channel receives errors from recv
	now := time.Now()
	ctx_, cancel := context.WithCancel(ctx)
	errors := make(chan error)

	// Open stream
	stream, err := this.GoogleCastClient.StreamEvents(ctx_, &empty.Empty{})
	if err != nil {
		return err
	}

	// Receive messages in the background, close when done
	go func() {
	FOR_LOOP:
		for {
			if evt_, err := stream.Recv(); err == io.EOF {
				break FOR_LOOP
			} else if err != nil {
				errors <- err
				break FOR_LOOP
			} else if evt := fromProtoEvent(evt_, this.RPCClientConn); evt != nil {
				now = time.Now()
				this.Emit(evt)
			}
		}
		close(errors)
	}()

	// Continue until error or io.EOF is returned, or nothing received after timeout
	heartbeat := time.NewTicker(HEARTBEAT_TIMEOUT)
	defer heartbeat.Stop()
	for {
		select {
		case <-heartbeat.C:
			if time.Since(now) > HEARTBEAT_TIMEOUT {
				// Haven't received a message for a while, so
				// register quit with deadline exceeded
				cancel()
			}
		case <-ctx_.Done():
			if err := <-errors; err == nil {
				return nil
			} else {
				return err
			}
		}
	}
}

////////////////////////////////////////////////////////////////////////////////
// STRINGIFY

func (this *Client) String() string {
	return fmt.Sprintf("<rpc.service.googlecast.Client>{ conn=%v }", this.RPCClientConn)
}
