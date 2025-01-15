// Access UEFI variables through the specialized [efivarfs] filesystem, which is typically mounted at /sys/firmware/efi/efivars.
//
// https://uefi.org/specs/UEFI/2.10_A/03_Boot_Manager.html#global-variables
//
// [efivarfs]: https://docs.kernel.org/filesystems/efivarfs.html
package efivarfs
