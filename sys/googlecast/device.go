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
	"strconv"
	"strings"
	"sync"

	// Frameworks
	gopi "github.com/djthorpe/gopi"
)

////////////////////////////////////////////////////////////////////////////////
// TYPES

type device struct {
	gopi.RPCServiceRecord
	sync.Mutex
	txt_ map[string]string
}

////////////////////////////////////////////////////////////////////////////////
// STRINGIFY

func (this *device) String() string {
	return fmt.Sprintf("<googlecast.Device>{ id=%v name=%v model=%v service=%v state=%v }", this.Id(), strconv.Quote(this.Name()), strconv.Quote(this.Model()), strconv.Quote(this.Service()), this.State())
}

////////////////////////////////////////////////////////////////////////////////
// PROPERTIES

func (this *device) Id() string {
	return this.txt("id")
}

func (this *device) Name() string {
	return this.txt("fn")
}

func (this *device) Model() string {
	return this.txt("md")
}

func (this *device) Service() string {
	return this.txt("rs")
}

func (this *device) State() uint {
	if value := this.txt("st"); value == "" {
		return 0
	} else if value_, err := strconv.ParseUint(value, 10, 32); err != nil {
		return 0
	} else {
		return uint(value_)
	}
}

func (this *device) Equals(other *device) bool {
	if this.Id() != other.Id() {
		return false
	}
	if this.Name() != other.Name() {
		return false
	}
	if this.Model() != other.Model() {
		return false
	}
	if this.Service() != other.Service() {
		return false
	}
	if this.State() != other.State() {
		return false
	}
	return true
}

////////////////////////////////////////////////////////////////////////////////
// PRIVATE METHODS

func (this *device) txt(key string) string {
	this.Mutex.Lock()
	defer this.Mutex.Unlock()
	if this.txt_ == nil {
		this.txt_ = make(map[string]string)
		for _, txt := range this.RPCServiceRecord.Text() {
			if pair := strings.SplitN(txt, "=", 2); len(pair) == 2 {
				this.txt_[pair[0]] = pair[1]
			}
		}
	}
	if value, exists := this.txt_[key]; exists {
		return value
	} else {
		return ""
	}
}
