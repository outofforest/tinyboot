package tinyboot

import (
	"context"
	"strings"

	"github.com/mdlayher/genetlink"
	"github.com/mdlayher/netlink"
)

// Action is the ACPI action to take
type Action int

const (
	// NoAction is returned when context is canceled or error occured
	NoAction Action = iota

	// PowerOffAction means system should be powered off
	PowerOffAction

	// RebootAction means system should be rebooted
	RebootAction
)

const (
	// See https://github.com/torvalds/linux/blob/master/drivers/acpi/event.c
	acpiGenlFamilyName     = "acpi_event"
	acpiGenlMcastGroupName = "acpi_mc_group"
)

// WaitACPI waits for ACPI signal
func WaitACPI(ctx context.Context) (Action, error) {
	// Get the acpi_event family.
	conn, err := genetlink.Dial(nil)
	if err != nil {
		return NoAction, err
	}
	defer conn.Close()

	f, err := conn.GetFamily(acpiGenlFamilyName)
	if err != nil {
		return NoAction, err
	}

	var id uint32
	for _, group := range f.Groups {
		if group.Name == acpiGenlMcastGroupName {
			id = group.ID
			break
		}
	}

	if err := conn.JoinGroup(id); err != nil {
		return NoAction, err
	}

	received := make(chan Action)
	errCh := make(chan error, 1)
	go func() {
		for {
			msgs, _, err := conn.Receive()
			if err != nil {
				errCh <- err
				return
			}

			if len(msgs) > 0 {
				action, err := parse(msgs)
				if err != nil {
					continue
				}

				switch action {
				case NoAction:
				default:
					received <- action
					return
				}
			}
		}
	}()

	select {
	case <-ctx.Done():
	case action := <-received:
		return action, nil
	case err := <-errCh:
		return NoAction, err
	}
	return NoAction, nil
}

func parse(msgs []genetlink.Message) (Action, error) {
	for _, msg := range msgs {
		ad, err := netlink.NewAttributeDecoder(msg.Data)
		if err != nil {
			return NoAction, err
		}

		for ad.Next() {
			if strings.HasPrefix(ad.String(), "button/power") {
				switch ad.Bytes()[40] {
				case 0x1:
					return PowerOffAction, nil
				case 0x2:
					return RebootAction, nil
				default:
					return NoAction, nil
				}
			}
		}
	}
	return NoAction, nil
}
