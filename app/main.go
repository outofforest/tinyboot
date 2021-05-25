package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
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

func main() {
	// ignore logs produced by dhclient
	log.SetOutput(ioutil.Discard)

	fmt.Println("Loading kernel modules")
	ok(modprobe("/modules/failover.ko"))
	ok(modprobe("/modules/net_failover.ko"))
	ok(modprobe("/modules/virtio_net.ko"))
	ok(modprobe("/modules/virtio_scsi.ko"))
	ok(modprobe("/modules/virtio_blk.ko"))

	fmt.Println("Network interfaces:")
	ifs, err := net.Interfaces()
	ok(err)

	wait := sync.WaitGroup{}
	for _, iface := range ifs {
		name := iface.Name
		fmt.Println(name)
		if iface.Flags&net.FlagLoopback != 0 {
			// loopback interface
			continue
		}

		wait.Add(1)

		link, err := tenus.NewLinkFrom(name)
		ok(err)
		if iface.Flags&net.FlagUp == 0 {
			fmt.Printf("Interface %s is down, starting it\n", name)
			ok(link.SetLinkUp())
		}
		fmt.Printf("Setting DHCP for interface %s\n", name)
		client := dhclient.Client{
			Iface: &iface,
			OnBound: func(lease *dhclient.Lease) {
				ok(link.SetLinkIp(lease.FixedAddress, &net.IPNet{IP: lease.FixedAddress, Mask: lease.Netmask}))
				ok(link.SetLinkDefaultGw(&lease.Router[0]))
				fmt.Printf("Interface %s configured:\n%+v\n", name, lease)

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

	for {
		func() {
			resp, err := http.Get("https://www.google.com")
			if err != nil {
				fmt.Println(err)
			} else {
				defer resp.Body.Close()
				_, err = io.Copy(os.Stdout, resp.Body)
				ok(err)
			}
		}()
		<-time.After(time.Second)
	}
}

func modprobe(file string) error {
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()

	return unix.FinitModule(int(f.Fd()), "", 0)
}

func ok(err error) {
	if err != nil {
		panic(err)
	}
}
