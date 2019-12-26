/*
  Go Language Raspberry Pi Interface
  (c) Copyright David Thorpe 2019
  All Rights Reserved
  Documentation http://djthorpe.github.io/gopi/
  For Licensing and Usage information, please see LICENSE.md
*/

package googlecast

import (
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
	CAST_EVENT_NONE EventType = iota
	CAST_EVENT_DEVICE_ADDED
	CAST_EVENT_DEVICE_UPDATED
	CAST_EVENT_DEVICE_DELETED
	CAST_EVENT_CHANNEL_CONNECT
	CAST_EVENT_CHANNEL_DISCONNECT
	CAST_EVENT_VOLUME_UPDATED
	CAST_EVENT_APPLICATION_UPDATED
	CAST_EVENT_MEDIA_UPDATED
)

////////////////////////////////////////////////////////////////////////////////
// INTERFACES

type Cast interface {
	gopi.Driver
	gopi.Publisher

	// Return list of discovered Google Chromecast Devices
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
	Application() Application
	Volume() Volume
	Media() Media

	/*
		// Set Properties
		SetApplication(Application) error // Application to watch or nil
		SetPlay(bool) (int, error)        // Play or stop
		SetPause(bool) (int, error)       // Pause or play
		SetVolume(float32) (int, error)   // Set volume level
		SetMuted(bool) (int, error)       // Set muted
		//SetTrackNext() (int, error)
		//SetTrackPrev() (int, error)
		//StreamUrl(string)
	*/
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
	//Devices() ([]Device, error)

	// Stream discovery events
	//StreamEvents(ctx context.Context) error
}

////////////////////////////////////////////////////////////////////////////////
// STRINGIFY

func (t EventType) String() string {
	switch t {
	case CAST_EVENT_NONE:
		return "CAST_EVENT_NONE"
	case CAST_EVENT_DEVICE_ADDED:
		return "CAST_EVENT_DEVICE_ADDED"
	case CAST_EVENT_DEVICE_UPDATED:
		return "CAST_EVENT_DEVICE_UPDATED"
	case CAST_EVENT_DEVICE_DELETED:
		return "CAST_EVENT_DEVICE_DELETED"
	case CAST_EVENT_CHANNEL_CONNECT:
		return "CAST_EVENT_CHANNEL_CONNECT"
	case CAST_EVENT_CHANNEL_DISCONNECT:
		return "CAST_EVENT_CHANNEL_DISCONNECT"
	case CAST_EVENT_VOLUME_UPDATED:
		return "CAST_EVENT_VOLUME_UPDATED"
	case CAST_EVENT_APPLICATION_UPDATED:
		return "CAST_EVENT_APPLICATION_UPDATED"
	case CAST_EVENT_MEDIA_UPDATED:
		return "CAST_EVENT_MEDIA_UPDATED"
	default:
		return "[?? Invalid GoogleCastEventType value]"
	}
}
