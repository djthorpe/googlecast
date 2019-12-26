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

	// Frameworks
	googlecast "github.com/djthorpe/googlecast"
	gopi "github.com/djthorpe/gopi"
	grpc "github.com/djthorpe/gopi-rpc/sys/grpc"

	// Protocol buffers
	pb "github.com/djthorpe/googlecast/rpc/protobuf/googlecast"
	empty "github.com/golang/protobuf/ptypes/empty"
)

////////////////////////////////////////////////////////////////////////////////
// TYPES

type Service struct {
	Server gopi.RPCServer
	Cast   googlecast.Cast
}

type service struct {
	log  gopi.Logger
	cast googlecast.Cast
}

////////////////////////////////////////////////////////////////////////////////
// OPEN AND CLOSE

// Open the server
func (config Service) Open(log gopi.Logger) (gopi.Driver, error) {
	log.Debug("<grpc.service.googlecast>Open{ %+v }", config)

	if config.Server == nil || config.Cast == nil {
		return nil, gopi.ErrBadParameter
	}

	this := new(service)
	this.log = log
	this.cast = config.Cast

	// Register service with GRPC server
	pb.RegisterGoogleCastServer(config.Server.(grpc.GRPCServer).GRPCServer(), this)

	// Success
	return this, nil
}

func (this *service) Close() error {
	this.log.Debug("<grpc.service.googlecast>Close{}")

	// Success
	return nil
}

////////////////////////////////////////////////////////////////////////////////
// RPCService implementation

func (this *service) CancelRequests() error {
	this.log.Debug("<grpc.service.googlecast>CancelRequests{}")
	return nil
}

////////////////////////////////////////////////////////////////////////////////
// Stringify

func (this *service) String() string {
	return fmt.Sprintf("<grpc.service.googlecast>{ %v }", this.cast)
}

////////////////////////////////////////////////////////////////////////////////
// RPC Methods

func (this *service) Ping(context.Context, *empty.Empty) (*empty.Empty, error) {
	this.log.Debug("<grpc.service.googlecast.Ping>{ }")
	return &empty.Empty{}, nil
}

func (this *service) Devices(context.Context, *empty.Empty) (*pb.DevicesReply, error) {
	this.log.Debug("<grpc.service.googlecast.Devices>{ }")
	return toProtoDevicesReply(this.cast.Devices()), nil
}
