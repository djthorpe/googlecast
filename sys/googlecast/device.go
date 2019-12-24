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
	"math/rand"
	"net"
	"strconv"
	"strings"
	"sync"

	// Frameworks
	gopi "github.com/djthorpe/gopi"
)

////////////////////////////////////////////////////////////////////////////////
// TYPES

type castdevice struct {
	gopi.RPCServiceRecord
	sync.Mutex
	txt_ map[string]string
}

////////////////////////////////////////////////////////////////////////////////
// STRINGIFY

func (this *castdevice) String() string {
	return fmt.Sprintf("<googlecast.Device>{ id=%v name=%v model=%v service=%v state=%v }", this.Id(), strconv.Quote(this.Name()), strconv.Quote(this.Model()), strconv.Quote(this.Service()), this.State())
}

////////////////////////////////////////////////////////////////////////////////
// PROPERTIES

func (this *castdevice) Id() string {
	return this.txt("id")
}

func (this *castdevice) Name() string {
	return this.txt("fn")
}

func (this *castdevice) Model() string {
	return this.txt("md")
}

func (this *castdevice) Service() string {
	return this.txt("rs")
}

func (this *castdevice) State() uint {
	if value := this.txt("st"); value == "" {
		return 0
	} else if value_, err := strconv.ParseUint(value, 10, 32); err != nil {
		return 0
	} else {
		return uint(value_)
	}
}

func (this *castdevice) Equals(other *castdevice) bool {
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

func (this *castdevice) txt(key string) string {
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

func (this *castdevice) addr(flag gopi.RPCFlag) (net.IP, error) {
	switch flag & (gopi.RPC_FLAG_INET_V4 | gopi.RPC_FLAG_INET_V6) {
	case gopi.RPC_FLAG_INET_V4:
		ip4 := this.RPCServiceRecord.IP4()
		if len(ip4) == 0 {
			return nil, gopi.ErrNotFound
		} else if flag&gopi.RPC_FLAG_SERVICE_ANY != 0 {
			// Return any
			index := rand.Intn(len(ip4) - 1)
			return ip4[index], nil
		} else {
			// Return first
			return ip4[0], nil
		}
	case gopi.RPC_FLAG_INET_V6:
		ip6 := this.RPCServiceRecord.IP6()
		if len(ip6) == 0 {
			return nil, gopi.ErrNotFound
		} else if flag&gopi.RPC_FLAG_SERVICE_ANY != 0 {
			// Return any
			index := rand.Intn(len(ip6) - 1)
			return ip6[index], nil
		} else {
			// Return first
			return ip6[0], nil
		}
	case (gopi.RPC_FLAG_INET_V6 | gopi.RPC_FLAG_INET_V4):
		if addr, err := this.addr(gopi.RPC_FLAG_INET_V4); err == nil {
			return addr, nil
		} else if addr, err := this.addr(gopi.RPC_FLAG_INET_V6); err == nil {
			return addr, nil
		} else {
			return nil, gopi.ErrNotFound
		}
	default:
		return nil, gopi.ErrBadParameter
	}
}
