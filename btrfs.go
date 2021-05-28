package tinyboot

import (
	"io/ioutil"
	"os"
)

// We automatically mount btrfs partition labeled "tinyboot"
// Btrfs format documentation: https://btrfs.wiki.kernel.org/index.php/On-disk_Format
// Superblock starts at offset 0x10000 (64 KiB)
// At offset 0x40 of superblock magic string "_BHRfS_M" exists
// At offset 0x12b label exists - length: 0x100 bytes

const btrfsID = "_BHRfS_M"

func findDrive(label string) string {
	devs, err := ioutil.ReadDir("/sys/class/block")
	if err != nil {
		panic(err)
	}

	for _, dev := range devs {
		if dev.IsDir() {
			continue
		}

		path := "/dev/" + dev.Name()
		f, err := os.Open(path)
		if err != nil {
			continue
		}
		defer f.Close()

		id := make([]byte, 0x8)
		if _, err := f.ReadAt(id, 0x10040); err != nil {
			continue
		}
		if string(id) != btrfsID {
			continue
		}

		labelRaw := make([]byte, 0x100)
		if _, err := f.ReadAt(labelRaw, 0x1012b); err != nil {
			continue
		}
		for i, ch := range labelRaw {
			if ch == 0x0 {
				labelRaw = labelRaw[:i]
				break
			}
		}
		if string(labelRaw) == label {
			return path
		}
	}
	return ""
}
