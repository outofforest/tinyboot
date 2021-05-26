# Tinyboot

This is an experimental project created to test how
custom tiny initramfs may be used to run go applications.

`build.sh` script builds go application stored in `/app`
and places it inside initramfs together with files required
to support networking inside qemu.

Then bootable ISO is created containing this initramfs and kernel.
Boot stub feature of kernel is used so no separate bootloader is installed.

GO app is responsible for loading network drivers,
configuring interfaces using DHCP and setting DNS resolver.

After doing those things we have fully-operable GO application
connected to the internet packed into small ~15MB ISO file.

`build.sh` creates initramfs based on Fedora 34.

## Missing features
- clock synchronization
- mounting persistent storage
- mounting cdrom to offload initramfs (see thoughts below)

## Thoughts
- Keeping big apps inside initramfs is not perfect because 
  it eats twice more RAM: first time as a part of initramfs, second time
  when binary is loaded and executed. It's better to store main app inside ISO
  image next to initramfs, mount /dev/cdrom and execute the main app from there.