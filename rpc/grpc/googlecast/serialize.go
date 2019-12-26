/*
	Go Language Raspberry Pi Interface
	(c) Copyright David Thorpe 2019
	All Rights Reserved
	Documentation http://djthorpe.github.io/gopi/
	For Licensing and Usage information, please see LICENSE.md
*/

package googlecast

import (
	// Frameworks
	"fmt"
	"strconv"

	googlecast "github.com/djthorpe/googlecast"

	// Protocol buffers
	pb "github.com/djthorpe/googlecast/rpc/protobuf/googlecast"
)

////////////////////////////////////////////////////////////////////////////////
// TYPES

type castdevice struct {
	*pb.CastDevice
}

////////////////////////////////////////////////////////////////////////////////
// DEVICE IMPLEMENTATION

func (this *castdevice) Id() string {
	if this.CastDevice == nil {
		return ""
	} else {
		return this.GetId()
	}
}

func (this *castdevice) Name() string {
	if this.CastDevice == nil {
		return ""
	} else {
		return this.GetName()
	}
}
func (this *castdevice) Model() string {
	if this.CastDevice == nil {
		return ""
	} else {
		return this.GetModel()
	}
}

func (this *castdevice) Service() string {
	if this.CastDevice == nil {
		return ""
	} else {
		return this.GetService()
	}
}

func (this *castdevice) State() uint {
	if this.CastDevice == nil {
		return 0
	} else {
		return uint(this.GetState())
	}
}

func (this *castdevice) String() string {
	if this == nil {
		return "<googlecast.Device>{ nil }"
	} else {
		return fmt.Sprintf("<googlecast.Device>{ id=%v name=%v model=%v service=%v state=%v }",
			strconv.Quote(this.Id()),
			strconv.Quote(this.Name()),
			strconv.Quote(this.Model()),
			strconv.Quote(this.Service()),
			this.State(),
		)
	}
}

////////////////////////////////////////////////////////////////////////////////
// FROM PROTO

func fromProtoDevicesReply(pb *pb.DevicesReply) []googlecast.Device {
	if pb == nil {
		return nil
	}
	devices := make([]googlecast.Device, len(pb.Device))
	for i, device := range pb.Device {
		devices[i] = &castdevice{device}
	}
	return devices
}

////////////////////////////////////////////////////////////////////////////////
// TO PROTO

func toProtoDevicesReply(devices []googlecast.Device) *pb.DevicesReply {
	if devices == nil {
		return nil
	}
	reply := &pb.DevicesReply{
		Device: make([]*pb.CastDevice, len(devices)),
	}
	for i, device := range devices {
		reply.Device[i] = &pb.CastDevice{
			Id:      device.Id(),
			Name:    device.Name(),
			Model:   device.Model(),
			Service: device.Service(),
			State:   uint32(device.State()),
		}
	}
	return reply
}
