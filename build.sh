#!/bin/sh

set -e

TMP=/tmp
mkdir -p "$TMP"

ISO_EFI=$(mktemp "$TMP"/iso-server-tmp.XXXXXX -d)
ISO_CONTENT=$(mktemp "$TMP"/iso-server-content.XXXXXX -d)
INITRAMFS=$(mktemp "$TMP"/iso-server-initramfs.XXXXXX -d)
MODULES=/lib/modules/$(uname -r)

dd if=/dev/zero of="$ISO_CONTENT"/efi.img bs=7M seek=1 count=0
LOOP_DEV=`losetup -f --show "$ISO_CONTENT"/efi.img`
mkfs.vfat "$LOOP_DEV"
mount "$LOOP_DEV" "$ISO_EFI"

cp -a /boot/efi/EFI "$ISO_EFI"

umount "$ISO_EFI"
losetup -d "$LOOP_DEV"

cp /boot/vmlinuz-*.x86_64 "$ISO_CONTENT"/vmlinuz

CGO_ENABLED=0 go build -o "$INITRAMFS"/init ./app

mkdir -p "$INITRAMFS"/{modules,etc/pki/tls/certs}

xzcat "$MODULES"/kernel/drivers/block/virtio_blk.ko.xz > "$INITRAMFS"/modules/virtio_blk.ko
xzcat "$MODULES"/kernel/drivers/scsi/virtio_scsi.ko.xz > "$INITRAMFS"/modules/virtio_scsi.ko
xzcat "$MODULES"/kernel/drivers/net/virtio_net.ko.xz > "$INITRAMFS"/modules/virtio_net.ko
xzcat "$MODULES"/kernel/drivers/net/net_failover.ko.xz > "$INITRAMFS"/modules/net_failover.ko
xzcat "$MODULES"/kernel/net/core/failover.ko.xz > "$INITRAMFS"/modules/failover.ko

cp /etc/pki/tls/certs/ca-bundle.crt "$INITRAMFS"/etc/pki/tls/certs

pushd "$INITRAMFS"
find . | cpio -c -o --owner root:root | xz --check=crc32 > "$ISO_CONTENT"/initramfs.img
popd

mkdir -p "$ISO_CONTENT"/EFI/fedora
cat > "$ISO_CONTENT"/EFI/fedora/grub.cfg << EOF
set default=0

insmod part_gpt
insmod fat

set timeout=1

menuentry 'Start OS' {
  linuxefi /vmlinuz loglevel=4 #console=tty0 console=ttyS0,9600n8
  initrdefi /initramfs.img
}
EOF

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
