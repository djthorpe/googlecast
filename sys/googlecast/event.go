/*
  Go Language Raspberry Pi Interface
  (c) Copyright David Thorpe 2019
  All Rights Reserved
  Documentation http://djthorpe.github.io/gopi/
  For Licensing and Usage information, please see LICENSE.md
*/

package googlecast

import (
	"fmt"

	// Frameworks
	googlecast "github.com/djthorpe/googlecast"
	gopi "github.com/djthorpe/gopi"
)

////////////////////////////////////////////////////////////////////////////////
// TYPES

type castevent struct {
	type_    googlecast.EventType
	source_  gopi.Driver
	device_  googlecast.Device
	channel_ googlecast.Channel
	reqid_   int
}

////////////////////////////////////////////////////////////////////////////////
// IMPLEMENTATION

func (castevent) Name() string {
	return "GooglecastEvent"
}

func (this *castevent) Source() gopi.Driver {
	return this.source_
}

func (this *castevent) Type() googlecast.EventType {
	return this.type_
}

func (this *castevent) Device() googlecast.Device {
	return this.device_
}

func (this *castevent) Channel() googlecast.Channel {
	return this.channel_
}

////////////////////////////////////////////////////////////////////////////////
// STRINGIFY

func (this *castevent) String() string {
	if this.channel_ != nil {
		return fmt.Sprintf("<%s>{ %v channel=%v device=%v reqid=%v }", this.Name(), this.type_, this.channel_, this.device_, this.reqid_)
	} else if this.device_ != nil {
		return fmt.Sprintf("<%s>{ %v device=%v }", this.Name(), this.type_, this.device_)
	} else {
		return fmt.Sprintf("<%s>{ %v }", this.Name(), this.type_)
	}
}
