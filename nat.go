// Package nat implements NAT handling facilities
package nat

import (
	"errors"
	"math"
	"math/rand"
	"net"
	"time"
)

var errNoAvailableExternalPort = errors.New("failed to find available external port")
var errExternalPortInUse = errors.New("external port is mapped to another internal port")
var errNoExternalAddress = errors.New("no external address")
var errNoInternalAddress = errors.New("no internal address")
var errNoNATFound = errors.New("no NAT found")

// NAT is the nat interface
type NAT interface {
	// Type returns the kind of NAT port mapping service that is used
	Type() string

	// GetDeviceAddress returns the internal address of the gateway device.
	GetDeviceAddress() (addr net.IP, err error)

	// GetExternalAddress returns the external address of the gateway device.
	GetExternalAddress() (addr net.IP, err error)

	// GetInternalAddress returns the address of the local host.
	GetInternalAddress() (addr net.IP, err error)

	// AddPortMapping maps a port on the local host to an external port.
	// protocol is either "udp" or "tcp"
	AddPortMapping(protocol string, externalPort int, internalPort int, description string, timeout time.Duration) (mappedExternalPort, mappedInternalPort int, err error)

	// DeletePortMapping removes a port mapping.
	DeletePortMapping(protocol string, externalPort int) (err error)
}

// DiscoverGateway attempts to find a gateway device.
func DiscoverGateway() (NAT, error) {
	select {
	case nat := <-discoverUPNPIG1():
		return nat, nil
	case nat := <-discoverUPNPIG2():
		return nat, nil
	case nat := <-discoverUPNP_GenIGDev():
		return nat, nil
	case nat := <-discoverNATPMP():
		return nat, nil
	case <-time.After(10 * time.Second):
		return nil, errNoNATFound
	}
}

func randomPort() int {
	rand.Seed(time.Now().UnixNano())
	return rand.Intn(math.MaxUint16-10000) + 10000
}
