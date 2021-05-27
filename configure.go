package tinyboot

import (
	"context"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/digineo/go-dhclient"
	"github.com/google/gopacket/layers"
	"github.com/milosgajdos/tenus"
	"golang.org/x/sys/unix"
)

var dhcpRequestList = []layers.DHCPOpt{
	layers.DHCPOptSubnetMask,
	layers.DHCPOptRouter,
}

// Configure configures tinyboot os
func Configure() {
	if os.Args[0] != "/init" {
		// We are not inside tinyboot os so there is nothing to configure
		return
	}

	// Load kernel modules for qemu hardware
	modprobe("/modules/failover.ko")
	modprobe("/modules/net_failover.ko")
	modprobe("/modules/virtio_net.ko")

	// Configure network
	ifs, err := net.Interfaces()
	ok(err)

	wait := sync.WaitGroup{}
	for _, iface := range ifs {
		name := iface.Name
		if iface.Flags&net.FlagLoopback != 0 {
			// loopback interface
			continue
		}

		wait.Add(1)

		link, err := tenus.NewLinkFrom(name)
		ok(err)
		if iface.Flags&net.FlagUp == 0 {
			ok(link.SetLinkUp())
		}
		client := dhclient.Client{
			Iface: &iface,
			OnBound: func(lease *dhclient.Lease) {
				ok(link.SetLinkIp(lease.FixedAddress, &net.IPNet{IP: lease.FixedAddress, Mask: lease.Netmask}))
				ok(link.SetLinkDefaultGw(&lease.Router[0]))

				wait.Done()
			},
		}

		for _, param := range dhcpRequestList {
			client.AddParamRequest(param)
		}

		client.Start()
		defer client.Stop()
	}
	wait.Wait()

	// Configure DNS resolver
	dialer := &net.Dialer{
		Resolver: &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{
					Timeout: 5 * time.Second,
				}
				return d.DialContext(ctx, "udp", "8.8.8.8:53")
			},
		},
	}

	http.DefaultTransport.(*http.Transport).DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		return dialer.DialContext(ctx, network, addr)
	}
}

func modprobe(file string) {
	f, err := os.Open(file)
	ok(err)
	defer func() {
		ok(f.Close())
	}()

	ok(unix.FinitModule(int(f.Fd()), "", 0))
}

func ok(err error) {
	if err != nil {
		panic(err)
	}
}
