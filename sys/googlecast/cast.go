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
	"sync"
	"time"

	// Frameworks
	gopi "github.com/djthorpe/gopi"
	errors "github.com/djthorpe/gopi/util/errors"
	event "github.com/djthorpe/gopi/util/event"
)

////////////////////////////////////////////////////////////////////////////////
// TYPES

type Cast struct {
	Discovery gopi.RPCServiceDiscovery
}

type cast struct {
	log       gopi.Logger
	discovery gopi.RPCServiceDiscovery
	event.Publisher
	event.Tasks
	sync.WaitGroup
}

////////////////////////////////////////////////////////////////////////////////
// COMSTANTS

const (
	SERVICE_TYPE_GOOGLECAST = "_googlecast._tcp"
	DELTA_LOOKUP_TIME       = 60 * time.Second
)

////////////////////////////////////////////////////////////////////////////////
// OPEN AND CLOSE

func (config Cast) Open(logger gopi.Logger) (gopi.Driver, error) {
	logger.Debug("<googlecast.Open>{ discovery=%v }", config.Discovery)

	this := new(cast)
	this.log = logger
	this.discovery = config.Discovery

	if this.discovery == nil {
		return nil, gopi.ErrBadParameter
	}

	// Success
	return this, nil
}

func (this *cast) Close() error {
	this.log.Debug("<googlecast.Close>{ }")

	errs := errors.CompoundError{}

	// Stop background tasks
	if err := this.Tasks.Close(); err != nil {
		errs.Add(err)
	}

	// Unsubscribe
	this.Publisher.Close()

	// Return any errors caught
	return errs.ErrorOrSelf()
}

////////////////////////////////////////////////////////////////////////////////
// STRINGIFY

func (this *cast) String() string {
	return fmt.Sprintf("<googlecast>{ }")
}
