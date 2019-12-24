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

	googlecast "github.com/djthorpe/googlecast"
	gopi "github.com/djthorpe/gopi"
)

////////////////////////////////////////////////////////////////////////////////
// TYPES

type castevent struct {
	type_   googlecast.EventType
	source_ gopi.Driver
	device_ googlecast.Device
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

////////////////////////////////////////////////////////////////////////////////
// STRINGIFY

func (this *castevent) String() string {
	return fmt.Sprintf("<%s>{ %v id=%v }", this.Name(), this.Type(), this.Device().Id())
}
