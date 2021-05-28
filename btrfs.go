package tinyboot

import (
	"io/ioutil"
	"os"
)

const btrfsID = "_BHRfS_M"

func findPersistentFS(label string) string {
	devs, err := ioutil.ReadDir("/sys/class/block")
	if err != nil {
		panic(err)
	}

	for _, dev := range devs {
		if dev.IsDir() {
			continue
		}

		if path := checkBtrfs(dev.Name(), label); path != "" {
			return path
		}
	}
	return ""
}

// Btrfs format documentation: https://btrfs.wiki.kernel.org/index.php/On-disk_Format
// Superblock starts at offset 0x10000 (64 KiB)
// At offset 0x40 of superblock magic string "_BHRfS_M" exists
// At offset 0x12b label exists - length: 0x100 bytes

func checkBtrfs(dev, label string) string {
	path := "/dev/" + dev
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	id := make([]byte, len(btrfsID))
	if _, err := f.ReadAt(id, 0x10040); err != nil {
		return ""
	}
	if string(id) != btrfsID {
		return ""
	}

	labelRaw := make([]byte, len(label))
	if _, err := f.ReadAt(labelRaw, 0x1012b); err != nil {
		return ""
	}
	if string(labelRaw) == label {
		return path
	}
	return ""
}
