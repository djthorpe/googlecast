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
	"time"

	// Frameworks
	"github.com/djthorpe/gopi"
)

////////////////////////////////////////////////////////////////////////////////
// TYPES

type (
	EventType uint
)

////////////////////////////////////////////////////////////////////////////////
// CONSTANTS

const (
	CAST_EVENT_NONE    EventType = 0
	CAST_EVENT_CONNECT EventType = iota
	CAST_EVENT_DISCONNECT
	CAST_EVENT_DEVICE
	CAST_EVENT_VOLUME
	CAST_EVENT_APPLICATION
	CAST_EVENT_MEDIA
)

////////////////////////////////////////////////////////////////////////////////
// INTERFACES

type Cast interface {
	gopi.Driver
	gopi.Publisher

	Devices() []Device

	// Connect to the control channel for a device, with timeout
	Connect(Device, gopi.RPCFlag, time.Duration) (Channel, error)
	Disconnect(Channel) error
}

type Device interface {
	Id() string
	Name() string
	Model() string
	Service() string
	State() uint
}

type Channel interface {
	// Address of channel
	RemoteAddr() string

	// Get Properties
	Applications() []Application
	Volume() Volume
	Media() Media

	// Set Properties
	SetApplication(Application) error // Application to watch or nil
	SetPlay(bool) (int, error)        // Play or stop
	SetPause(bool) (int, error)       // Pause or play
	SetVolume(float32) (int, error)   // Set volume level
	SetMuted(bool) (int, error)       // Set muted
	//SetTrackNext() (int, error)
	//SetTrackPrev() (int, error)
	//StreamUrl(string)
}

type Application interface {
	ID() string
	Name() string
	Status() string
}

type Volume interface {
	Level() float32
	Muted() bool
}

type Media interface {
}

type Event interface {
	gopi.Event

	Type() EventType
	Device() Device
	Channel() Channel
}

////////////////////////////////////////////////////////////////////////////////
// RPC CLIENT

type Client interface {
	gopi.RPCClient
	gopi.Publisher

	// Ping remote service
	Ping() error

	// Return devices from the remote service
	Devices() ([]Device, error)

	// Stream discovery events
	StreamEvents(ctx context.Context) error
}

////////////////////////////////////////////////////////////////////////////////
// STRINGIFY

func (t EventType) String() string {
	switch t {
	case CAST_EVENT_NONE:
		return "CAST_EVENT_NONE"
	case CAST_EVENT_CONNECT:
		return "CAST_EVENT_CONNECT"
	case CAST_EVENT_DISCONNECT:
		return "CAST_EVENT_DISCONNECT"
	case CAST_EVENT_DEVICE:
		return "CAST_EVENT_DEVICE"
	case CAST_EVENT_VOLUME:
		return "CAST_EVENT_VOLUME"
	case CAST_EVENT_APPLICATION:
		return "CAST_EVENT_APPLICATION"
	case CAST_EVENT_MEDIA:
		return "CAST_EVENT_MEDIA"
	default:
		return "[?? Invalid GoogleCastEventType value]"
	}
}
