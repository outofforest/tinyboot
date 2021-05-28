# Tinyboot

This project allows you to build tiny bootable ISO running GO application.
ISO built from this example takes ~14MB where ~2MB is taken by GO app itself.

`build.sh` script builds example GO application stored in `/app`
and places it inside initramfs together with files required
to support storage and networking inside qemu, using virtio drivers.

Then bootable ISO is created containing this initramfs and kernel.
Boot stub feature of kernel is used so no separate bootloader is required, saving space.

`build.sh` creates initramfs based on Fedora 34.

## Networking

System supports loopback interface and DHCP configuration of all recognized network
adapters. Only virtio driver is loaded currently. If you need sth else feel free to
copy and load appropriate module.

## Storage

Virtio driver is loaded to support storage delivered by qemu. If you need sth else, add
and load appropriate driver.

Supported filesystems:
- tmpfs
- ramfs
- ext2
- ext3
- ext4
- btrfs
- iso9660

## ISO content

To save RAM, you don't need to copy all the required content to initramfs.
`/dev` is scanned to find iso9660 filesystem. The first one found is mounted to `/iso`.
If your app requires read-only access to some files, you may copy them to ISO image and read from
there directly.

## Persistent storage

To support persistent storage, `/dev` is scanned to find drive containing `btrfs` filesystem
labeled `tinyboot`. The first one found is mounted to `/persistent`.

## ACPI

`tinyboot.Configure()` returns context which is canceled whenever machine is requested
to be turned off or rebooted and cleanup function which should be deferred and called before exit.
If application exits on its own (without signal from ACPI), deferred cleanup function causes reboot.
Same happens in case of panic.

Using ACPI functions (power off or restart option of VM) is the correct way to execute graceful application
shutdown or restart.

## Missing features
- clock synchronization
