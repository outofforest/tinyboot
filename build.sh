#!/bin/sh

set -e

TMP=/tmp
mkdir -p "$TMP"

ISO_EFI=$(mktemp "$TMP"/iso-server-tmp.XXXXXX -d)
ISO_CONTENT=$(mktemp "$TMP"/iso-server-content.XXXXXX -d)
INITRAMFS=$(mktemp "$TMP"/iso-server-initramfs.XXXXXX -d)
MODULES=/lib/modules/$(uname -r)

dd if=/dev/zero of="$ISO_CONTENT"/efi.img bs=14M seek=1 count=0
LOOP_DEV=`losetup -f --show "$ISO_CONTENT"/efi.img`
mkfs.vfat "$LOOP_DEV"
mount "$LOOP_DEV" "$ISO_EFI"

cp /boot/vmlinuz-*.x86_64 "$ISO_EFI"/vmlinuz

CGO_ENABLED=0 go build -o "$INITRAMFS"/init ./app

mkdir -p "$INITRAMFS"/{modules,etc/pki/tls/certs}

xzcat "$MODULES"/kernel/drivers/block/virtio_blk.ko.xz > "$INITRAMFS"/modules/virtio_blk.ko
xzcat "$MODULES"/kernel/drivers/scsi/virtio_scsi.ko.xz > "$INITRAMFS"/modules/virtio_scsi.ko
xzcat "$MODULES"/kernel/drivers/net/virtio_net.ko.xz > "$INITRAMFS"/modules/virtio_net.ko
xzcat "$MODULES"/kernel/drivers/net/net_failover.ko.xz > "$INITRAMFS"/modules/net_failover.ko
xzcat "$MODULES"/kernel/net/core/failover.ko.xz > "$INITRAMFS"/modules/failover.ko

cp /etc/pki/tls/certs/ca-bundle.crt "$INITRAMFS"/etc/pki/tls/certs

pushd "$INITRAMFS"
find . | cpio -c -o --owner root:root | xz --check=crc32 > "$ISO_EFI"/initramfs.img
popd

# startup.nsh is used as a fallback if no valid UEFi entry is found
echo "vmlinuz loglevel=4 initrd=\initramfs.img" > "$ISO_EFI"/startup.nsh

umount "$ISO_EFI"
losetup -d "$LOOP_DEV"

TIME=`date +"%Y-%m-%d-%H-%M-%S"`
ISO_OUT=./server-"$TIME".iso

xorriso -as mkisofs \
  -iso-level 3 \
  -r -V "os" \
  -J -joliet-long \
  -no-emul-boot \
  -e /efi.img \
  -partition_cyl_align all \
  -o "$ISO_OUT" \
  "$ISO_CONTENT"

rm -rf "$ISO_EFI" "$ISO_CONTENT" "$INITRAMFS"
