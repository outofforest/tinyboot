package tinyboot

import (
	"io/ioutil"
	"os"
)

const isofsID = "CD001"

func findCDROMFS() string {
	devs, err := ioutil.ReadDir("/sys/class/block")
	if err != nil {
		panic(err)
	}

	for _, dev := range devs {
		if dev.IsDir() {
			continue
		}

		if path := checkISO(dev.Name()); path != "" {
			return path
		}
	}
	return ""
}

// isofs format documentation: https://wiki.osdev.org/ISO_9660
// Volume descriptor starts at offset 0x8000 (32 KiB)
// At offset 0x1 of volume descriptor magic string "CD001" exists

func checkISO(dev string) string {
	path := "/dev/" + dev
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	id := make([]byte, len(isofsID))
	if _, err := f.ReadAt(id, 0x8001); err != nil {
		return ""
	}
	if string(id) == isofsID {
		return path
	}
	return ""
}
