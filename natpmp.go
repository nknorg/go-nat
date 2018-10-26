package nat

import (
	"net"
	"time"

	"github.com/jackpal/gateway"
	"github.com/jackpal/go-nat-pmp"
)

var (
	_ NAT = (*natpmpNAT)(nil)
)

func discoverNATPMP() <-chan NAT {
	res := make(chan NAT, 1)

	ip, err := gateway.DiscoverGateway()
	if err == nil {
		go discoverNATPMPWithAddr(res, ip)
	}

	return res
}

func discoverNATPMPWithAddr(c chan NAT, ip net.IP) {
	client := natpmp.NewClient(ip)
	_, err := client.GetExternalAddress()
	if err != nil {
		return
	}

	c <- &natpmpNAT{client, ip, make(map[int]int)}
}

type natpmpNAT struct {
	c       *natpmp.Client
	gateway net.IP
	ports   map[int]int
}

func (n *natpmpNAT) GetDeviceAddress() (addr net.IP, err error) {
	return n.gateway, nil
}

func (n *natpmpNAT) GetInternalAddress() (addr net.IP, err error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			return nil, err
		}

		for _, addr := range addrs {
			switch x := addr.(type) {
			case *net.IPNet:
				if x.Contains(n.gateway) {
					return x.IP, nil
				}
			}
		}
	}

	return nil, errNoInternalAddress
}

func (n *natpmpNAT) GetExternalAddress() (addr net.IP, err error) {
	res, err := n.c.GetExternalAddress()
	if err != nil {
		return nil, err
	}

	d := res.ExternalIPAddress
	return net.IPv4(d[0], d[1], d[2], d[3]), nil
}

func (n *natpmpNAT) AddPortMapping(protocol string, externalPort int, internalPort int, description string, timeout time.Duration) (int, int, error) {
	var (
		err error
	)

	timeoutInSeconds := int(timeout / time.Second)

	if externalPort == 0 {
		found := true
		for i := 0; i < 100; i++ {
			externalPort = randomPort()
			_, found = n.ports[externalPort]
			if !found {
				break
			}
		}
		if found {
			return 0, 0, errNoAvailableExternalPort
		}
	}

	if existingInternalPort, ok := n.ports[externalPort]; ok {
		if internalPort == 0 {
			internalPort = existingInternalPort
		}

		if internalPort != existingInternalPort {
			return 0, 0, errExternalPortInUse
		}

		_, err = n.c.AddPortMapping(protocol, internalPort, externalPort, timeoutInSeconds)
		if err != nil {
			return 0, 0, err
		}

		return externalPort, internalPort, nil
	}

	numTries := 1
	if internalPort == 0 {
		numTries = 3
	}

	for i := 0; i < numTries; i++ {
		if internalPort == 0 {
			_, err = n.c.AddPortMapping(protocol, randomPort(), externalPort, timeoutInSeconds)
		} else {
			_, err = n.c.AddPortMapping(protocol, internalPort, externalPort, timeoutInSeconds)
		}

		if err == nil {
			n.ports[externalPort] = internalPort
			return externalPort, internalPort, nil
		}
	}

	return 0, 0, err
}

func (n *natpmpNAT) DeletePortMapping(protocol string, externalPort int) (err error) {
	delete(n.ports, externalPort)
	return nil
}

func (n *natpmpNAT) Type() string {
	return "NAT-PMP"
}
