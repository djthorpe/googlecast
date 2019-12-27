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

type Service struct {
	Server gopi.RPCServer
	Cast   googlecast.Cast
}

type service struct {
	log  gopi.Logger
	cast googlecast.Cast
	event.Tasks
	event.Publisher
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

	// Start background thread
	this.Tasks.Start(this.EventsTask)

	// Success
	return this, nil
}

func (this *service) Close() error {
	this.log.Debug("<grpc.service.googlecast>Close{}")

	// Close events tasks
	if err := this.Tasks.Close(); err != nil {
		return err
	}

	// Success
	return nil
}

////////////////////////////////////////////////////////////////////////////////
// RPCService implementation

func (this *service) CancelRequests() error {
	this.log.Debug("<grpc.service.googlecast>CancelRequests{}")

	// Put empty event onto the channel to indicate any on-going
	// requests should be ended
	this.Emit(event.NullEvent)

	// Return success
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

// Stream events
func (this *service) StreamEvents(_ *empty.Empty, stream pb.GoogleCast_StreamEventsServer) error {
	this.log.Debug2("<grpc.service.googlecast.StreamEvents>{}")

	events := this.cast.Subscribe()
	cancel := this.Subscribe()
	ticker := time.NewTicker(time.Second)
FOR_LOOP:
	for {
		select {
		case evt := <-events:
			if evt == nil {
				break FOR_LOOP
			} else if evt_, ok := evt.(googlecast.Event); ok {
				fmt.Println(evt_)
				if err := stream.Send(toProtoEvent(evt_)); err != nil {
					this.log.Warn("StreamEvents: %v", err)
					break FOR_LOOP
				}
			} else {
				this.log.Warn("StreamEvents: Ignoring event: %v", evt)
			}
		case <-ticker.C:
			if err := stream.Send(&pb.CastEvent{}); err != nil {
				this.log.Warn("StreamEvents: %v", err)
				break FOR_LOOP
			}
		case <-cancel:
			break FOR_LOOP
		}
	}

	// Stop ticker, unsubscribe from events
	ticker.Stop()
	this.cast.Unsubscribe(events)
	this.Unsubscribe(cancel)

	this.log.Debug2("StreamEvents: Ended")

	// Return success
	return nil
}

////////////////////////////////////////////////////////////////////////////////
// BACKGROUND TASKS

func (this *service) EventsTask(start chan<- event.Signal, stop <-chan event.Signal) error {
	evt := this.cast.Subscribe()
	start <- gopi.DONE

FOR_LOOP:
	for {
		select {
		case event := <-evt:
			fmt.Println(event)
		case <-stop:
			this.cast.Unsubscribe(evt)
			break FOR_LOOP
		}
	}

	// Success
	return nil
}
