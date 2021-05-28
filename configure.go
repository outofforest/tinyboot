package tinyboot

import (
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/digineo/go-dhclient"
	"github.com/google/gopacket/layers"
	"github.com/milosgajdos/tenus"
	"golang.org/x/sys/unix"
)

const cdromMount = "/iso"
const persistentMount = "/persistent"
const persistentLabel = "tinyboot"

var dhcpRequestList = []layers.DHCPOpt{
	layers.DHCPOptSubnetMask,
	layers.DHCPOptRouter,
}

// Configure configures tinyboot os
func Configure() func() {
	if os.Args[0] != "/init" {
		// We are not inside tinyboot os so there is nothing to configure
		return func() {}
	}

	ready := make(chan struct{})
	exit := make(chan struct{})
	go func() {
		defer func() {
			mountsRaw, err := ioutil.ReadFile("/proc/mounts")
			ok(err)
			for _, mount := range strings.Split(string(mountsRaw), "\n") {
				props := strings.SplitN(mount, " ", 3)
				if len(props) < 2 {
					// last empty line
					break
				}
				mountpoint := props[1]
				switch mountpoint {
				case "/": // managed by vmlinuz
				case "/proc", "/sys", "/dev": // special mounts, unmounting them at the end
				default:
					ok(syscall.Unmount(mountpoint, 0))
				}
			}

			ok(syscall.Unmount("/proc", 0))
			ok(syscall.Unmount("/sys", 0))
			ok(syscall.Unmount("/dev", 0))

			ok(syscall.Reboot(syscall.LINUX_REBOOT_CMD_RESTART))
		}()

		// Load kernel modules for qemu network
		modprobe("/modules/failover.ko")
		modprobe("/modules/net_failover.ko")
		modprobe("/modules/virtio_net.ko")

		// Load kernel modules for qemu drives
		modprobe("/modules/virtio_blk.ko")
		modprobe("/modules/virtio_scsi.ko")

		// Load kernel modules for iso9660 filesystem
		modprobe("/modules/isofs.ko")

		// Mount filesystems

		ensure(os.Mkdir("/proc", 0o755))
		ensure(os.Mkdir("/sys", 0o755))
		ensure(os.Mkdir("/dev", 0o755))

		ok(syscall.Mount("none", "/proc", "proc", 0, ""))
		ok(syscall.Mount("none", "/sys", "sysfs", 0, ""))
		ok(syscall.Mount("none", "/dev", "devtmpfs", 0, ""))

		// Mount cdrom drive

		if drive := findCDROMFS(); drive != "" {
			ensure(os.Mkdir(cdromMount, 0o755))
			ok(syscall.Mount(drive, cdromMount, "iso9660", syscall.MS_RDONLY, ""))
		}

		// Mount persistent drive

		if drive := findPersistentFS(persistentLabel); drive != "" {
			ensure(os.Mkdir(persistentMount, 0o755))
			ok(syscall.Mount(drive, persistentMount, "btrfs", 0, ""))
		}

		// Configure network
		ifs, err := net.Interfaces()
		ok(err)

		wait := sync.WaitGroup{}
		for _, iface := range ifs {
			name := iface.Name
			link, err := tenus.NewLinkFrom(name)
			ok(err)
			if iface.Flags&net.FlagUp == 0 {
				ok(link.SetLinkUp())
			}

			if iface.Flags&net.FlagLoopback != 0 {
				// loopback interface
				continue
			}

			wait.Add(1)

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

		net.DefaultResolver = &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{
					Timeout: 5 * time.Second,
				}
				return d.DialContext(ctx, "udp", "8.8.8.8:53")
			},
		}

		close(ready)
		<-exit
	}()

	<-ready

	return func() {
		if p := recover(); p != nil {
			fmt.Printf("Application panicked: %v\n", p)
		}
		close(exit)

		// pause until reboot to prevent kernel panic
		<-make(chan struct{})
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

func ensure(err error) {
	if err != nil && !os.IsExist(err) {
		ok(err)
	}
}

func ok(err error) {
	if err != nil {
		fmt.Printf("Unexpected error: %s, will panic...", err.Error())
		<-time.After(5 * time.Second)
		panic(err)
	}
}
