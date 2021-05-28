#!/bin/sh

set -e

DIR=$(mktemp /tmp/iso-tinyboot.XXXXXX -d)
MODULES=/lib/modules/$(uname -r)

mkdir -p "$DIR"/{iso,efi,initramfs/modules,initramfs/etc/pki/tls/certs,initramfs/proc/self}

# Copy stuff to initramfs

CGO_ENABLED=0 go build -ldflags="-s -w" -o "$DIR"/initramfs/init ./app

# The /proc/self/exe path used by os.Executable is resolved at init time before procfs is mounted.
# To make it work fake /proc/self/exe has to be provided before starting GO application.
ln -s /init "$DIR"/initramfs/proc/self/exe

cp /etc/pki/tls/certs/ca-bundle.crt "$DIR"/initramfs/etc/pki/tls/certs # for trusted certs
xzcat "$MODULES"/kernel/drivers/block/virtio_blk.ko.xz > "$DIR"/initramfs/modules/virtio_blk.ko
xzcat "$MODULES"/kernel/drivers/scsi/virtio_scsi.ko.xz > "$DIR"/initramfs/modules/virtio_scsi.ko
xzcat "$MODULES"/kernel/drivers/net/virtio_net.ko.xz > "$DIR"/initramfs/modules/virtio_net.ko
xzcat "$MODULES"/kernel/drivers/net/net_failover.ko.xz > "$DIR"/initramfs/modules/net_failover.ko
xzcat "$MODULES"/kernel/net/core/failover.ko.xz > "$DIR"/initramfs/modules/failover.ko
xzcat "$MODULES"/kernel/fs/isofs/isofs.ko.xz > "$DIR"/initramfs/modules/isofs.ko

# Copy stuff to EFI

dd if=/dev/zero of="$DIR"/iso/efi.img bs=13M seek=1 count=0
mkfs.vfat "$DIR"/iso/efi.img
mount "$DIR"/iso/efi.img "$DIR"/efi

cp /boot/vmlinuz-*.x86_64 "$DIR"/efi/vmlinuz

pushd "$DIR"/initramfs
find . | cpio -c -o --owner root:root | xz --check=crc32 > "$DIR"/efi/initramfs.img
popd

# startup.nsh is used as a fallback if no valid UEFI entry is found
echo "vmlinuz loglevel=4 initrd=\initramfs.img" > "$DIR"/efi/startup.nsh

umount "$DIR"/efi

# Build ISO image

xorriso -as mkisofs \
  -iso-level 3 -r \
  -J -joliet-long \
  -no-emul-boot \
  -e /efi.img \
  -partition_cyl_align all \
  -o ./tinyboot-$(date +"%Y-%m-%d-%H-%M-%S").iso \
  "$DIR"/iso

rm -rf "$DIR"
