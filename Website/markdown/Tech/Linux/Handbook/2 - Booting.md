# Booting Linux

Before you install Linux, you need to understand how Linux gets from a file on
the internet to a running system on your machine. That usually means downloading
an ISO, writing it to a USB drive, booting from that USB drive, and deciding
whether you want to test, install, repair, or rescue something.

This sounds simple until your computer looks at the USB stick and says, "I have
decided this does not exist." That is when the fun begins.

## What Is An ISO?

An ISO is a disk image. Think of it like a complete snapshot of an installer,
live environment, or rescue system. You do not copy the ISO file onto a USB
drive like a normal document. You write the image to the USB drive in a special
way so the computer can boot from it.

Most Linux downloads come as `.iso` files:

```text
debian-12.5.0-amd64-netinst.iso
linuxmint-22-cinnamon-64bit.iso
archlinux-2026.05.01-x86_64.iso
```

The exact names change, but the idea is the same: this file becomes your
bootable installer or live system.

## Architectures

Before downloading an ISO, make sure you choose the right CPU architecture.

The most common one for normal desktops and laptops is:

```text
x86_64
amd64
```

Those usually mean the same thing: modern 64-bit Intel or AMD computers. Even
if your CPU is Intel, `amd64` is still probably the correct download. Yes, that
name is confusing. Computers are held together by history, caffeine, and
backward compatibility.

Other common architectures:

| Architecture | Usually Means |
| --- | --- |
| `amd64` / `x86_64` | Most modern Intel and AMD PCs |
| `arm64` / `aarch64` | ARM devices, some newer laptops, Raspberry Pi-class devices |
| `i386` / `x86` | Old 32-bit PCs |

If you are installing Linux on a normal modern desktop or laptop, pick
`amd64` or `x86_64`.

## Full ISO vs Netinstall ISO

Some distros offer different ISO types.

A full ISO includes more packages on the image. It is larger, but it can be
more useful if your internet connection is slow, limited, or unavailable during
installation.

A netinstall ISO is smaller. It boots into an installer and downloads many
packages from the internet during setup. Debian is famous for this style.

Use a full ISO when:

- You want more offline installation capability
- You have unreliable internet
- You want the desktop environment included on the USB

Use a netinstall ISO when:

- You have good internet
- You want a smaller download
- You want a cleaner minimal install

Neither is automatically better. They are tools. Pick the one that matches the
job instead of declaring loyalty to an installer format like it is a sports
team.

## Live ISOs

Many Linux distros provide a live ISO.

A live ISO lets you boot into Linux without installing it to the computer. You
can test hardware, connect to Wi-Fi, browse the desktop, open a terminal, and
make sure the system does not immediately hate your graphics card.

Live sessions are great for:

- Testing a distro before installing
- Checking Wi-Fi, audio, touchpad, GPU, and display support
- Recovering files from a broken system
- Running partition tools
- Doing quick troubleshooting

Most live sessions do not save your changes after reboot. You can install apps,
change settings, and make a mess, but once you reboot, it disappears. In this
case, disappearing is a feature, not a disaster.

## Persistent Live USBs

Some live USB setups support persistence.

Persistence means the USB can save changes between boots. Files, settings, and
sometimes installed packages can survive a restart.

This can be useful for:

- Portable toolkits
- Rescue USBs
- Testing Linux over multiple sessions
- Carrying a small personal environment

But persistence has limits. It can be slower than a real install, more fragile,
and easier to corrupt if the USB stick is cheap or removed at the wrong time.
It is useful, but I would not treat it like a permanent workstation unless you
enjoy living on the edge with a $7 flash drive.

## Ways To Run Linux

There are several ways to run Linux, and each has a different purpose.

### Live USB

Boot from a USB stick and test the system without installing.

Best for:

- Trying a distro
- Troubleshooting
- Recovering files
- Checking hardware support

### Full Install

Install Linux onto an internal drive or external SSD.

Best for:

- Daily use
- Better performance
- Long-term setup
- Full updates and customization

### Dual Boot

Install Linux alongside another operating system, usually Windows.

Best for:

- Keeping Windows for certain apps or games
- Learning Linux gradually
- Testing before fully switching

Dual booting is useful, but be careful. Back up your files first. Partitioning
is not the place to discover your inner optimist.

### Virtual Machine

Run Linux inside software like VirtualBox, VMware, GNOME Boxes, Hyper-V, or
UTM.

Best for:

- Learning safely
- Testing commands
- Trying distros quickly
- Avoiding partition changes

Virtual machines are one of the best ways to learn. You can break things,
delete the VM, and start over. This is much cheaper than breaking your real
machine and then learning emotional regulation.

## Writing An ISO To USB

To boot Linux from USB, you need to write the ISO image to the USB drive.

> [!WARNING]
> This usually erases the USB drive. Double-check the target drive before you
> write the image.

## Rufus On Windows

Rufus is my preferred tool on Windows.
https://rufus.ie/en/

It is fast, reliable, and gives you useful options without being too confusing.
For most Linux ISOs, the basic process is:

1. Insert the USB drive.
2. Open Rufus.
3. Select the ISO.
4. Select the correct USB device.
5. Click Start.

Rufus may ask whether to write in ISO mode or DD mode. ISO mode usually works
for many distros and keeps the USB easier for Windows to understand. DD mode
writes the image more directly and may be required by some distros.

If the distro documentation recommends one mode, follow the documentation. If
not, start with the default Rufus suggests.

## Ventoy

Ventoy is excellent if you want multiple ISOs on one USB drive.
https://www.ventoy.net/en/index.html

Instead of rewriting the USB every time, you install Ventoy to the USB once.
Then you copy ISO files onto the USB like normal files. When you boot, Ventoy
shows a menu and lets you choose which ISO to start.

Ventoy is great for:

- Carrying multiple Linux installers
- Keeping rescue tools in one place
- Testing many distros
- Avoiding constant USB rewriting

Example USB layout:

```text
ISOs/
  debian.iso
  linuxmint.iso
  archlinux.iso
  gparted-live.iso
  systemrescue.iso
```

Ventoy feels almost too convenient the first time you use it. That is allowed.
Sometimes the computer world lets us have one nice thing.

## dd On Linux

`dd` is the classic Linux command-line way to write an ISO to a USB drive.

It is powerful. It is also scary because it will happily overwrite the wrong
disk if you tell it to. `dd` does not ask if you are emotionally ready.

First, identify your USB drive:

```bash
lsblk
```

You might see something like:

```text
sda      931.5G disk
sdb       28.7G disk
```

If `sdb` is your USB drive, write the ISO like this:

```bash
sudo dd if=linux.iso of=/dev/sdX bs=4M status=progress conv=fsync
```

Replace `/dev/sdX` with the correct drive, such as `/dev/sdb`.

> [!DANGER]
> Do not write to a partition like `/dev/sdb1`. Write to the whole drive, like
> `/dev/sdb`. Also, do not guess. Use `lsblk` and verify the drive size.

After writing, you can run:

```bash
sync
```

Then safely remove the USB drive.

## Boot Menu Keys

To boot from USB, you usually need to open your computer's boot menu during
startup.

Common boot menu keys:

| Brand | Common Key |
| --- | --- |
| Dell | `F12` |
| HP | `Esc` or `F9` |
| Lenovo | `F12` or `Enter` |
| ASUS | `Esc` or `F8` |
| Acer | `F12` |
| MSI | `F11` |
| Gigabyte | `F12` |

These vary by model, because apparently one universal boot key would have made
too much sense.

If the boot menu does not show the USB drive:

- Try another USB port.
- Recreate the USB.
- Check whether Secure Boot is blocking it.
- Check whether the ISO supports UEFI.
- Make sure the USB was written as a bootable image, not copied as a file.

## UEFI, BIOS, And Secure Boot

Modern computers usually use UEFI instead of old BIOS.

UEFI is the firmware that starts before the operating system. It initializes
hardware, reads boot entries, and starts the bootloader.

Secure Boot is a UEFI feature that only allows trusted bootloaders to run. Some
distros support Secure Boot well. Some require extra setup. Some custom or
community ISOs may need Secure Boot disabled.

If your Linux USB will not boot, Secure Boot is one of the first things to
check.

General advice:

- Leave Secure Boot on if your distro supports it.
- Disable Secure Boot if the installer will not boot and the distro recommends
  doing so.
- Re-enable it later if your setup supports it and you want that protection.

Do not randomly change every firmware setting at once. Change one thing, test,
then continue. Firmware menus are where confidence goes to get humbled.

## Before Installing

Before installing Linux, do a few boring but important things.

- Back up important files.
- Confirm you downloaded the right ISO.
- Verify the USB boots.
- Test Wi-Fi and Ethernet in the live environment.
- Check whether your GPU works properly.
- Know whether you are wiping the disk or dual booting.
- If dual booting, make sure you understand the partition layout.

Boring preparation is much better than exciting data loss.

## A Simple Boot Workflow

Here is a practical flow:

1. Choose the distro.
2. Download the correct ISO for your architecture.
3. Optionally verify the checksum.
4. Write the ISO with Rufus, Ventoy, or `dd`.
5. Boot from the USB.
6. Test hardware in the live environment.
7. Decide whether to install, troubleshoot, or walk away with dignity.
8. Back up files before touching partitions.
9. Install Linux.
10. Reboot and remove the USB when prompted.

That is the basic path. The details change by distro, but the shape is almost
always the same.

## Final Thoughts

Booting Linux is not just a technical step. It is the doorway into the system.
Once you understand ISOs, architecture, live sessions, USB tools, boot menus,
and firmware settings, Linux installs become much less mysterious.

You will still occasionally meet a machine that refuses to boot for reasons
known only to its motherboard and whatever ancient firmware oath it took. But
most of the time, if you slow down and check each layer, the problem becomes
findable.
