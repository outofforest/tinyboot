# Tinyboot

This project allows you to build tiny bootable ISO running GO application.
ISO built from this example takes ~15MB where ~8MB is taken by GO app itself.

`build.sh` script builds example GO application stored in `/app`
and places it inside initramfs together with files required
to support storage and networking inside qemu, using virtio drivers.

Then bootable ISO is created containing this initramfs and kernel.
Boot stub feature of kernel is used so no separate bootloader is required, saving space.

`tinyboot.Configure()` call in the app is responsible for configuring the environment.

`build.sh` creates initramfs based on Fedora 34.

## Networking

System supports loopback interface and DHCP configuration of all recognized network
adapters. Only virtio driver is loaded currently. If you need sth else feel free to
copy and load appropriate module.

## Storage

Virtio driver is loaded to support storage delivered by qemu. If you need sth else, add
and load appropriate driver.

Filesystems supported by the kernel (at least mine):
- tmpfs
- ramfs
- ext2
- ext3
- ext4
- btrfs

## Mounting persistent storage

To support persistent storage, `/dev` is scanned to find drive containing `btrfs` filesystem
labeled `tinyboot`. The first one found is mounted to `/persistent`.

## Missing features
- clock synchronization
- ACPI signals for rebooting and powering off the machine
- mounting cdrom (iso image)
