// Copyright ZStack.io, Inc. 2013, 2026
// SPDX-License-Identifier: MPL-2.0

package utils

// MiBToBytes converts binary mebibytes to bytes.
func MiBToBytes(mib int64) int64 {
	return mib * 1024 * 1024
}

// MBToBytes is retained for compatibility; its unit is MiB.
func MBToBytes(mb int64) int64 { return MiBToBytes(mb) }

func BytesToMB(bytes int64) int64 {
	return bytes / (1024 * 1024)
}

func BytesToGB(bytes int64) int64 {
	return bytes / (1024 * 1024 * 1024)
}

func GBToBytes(gb int64) int64 {
	return gb * 1024 * 1024 * 1024
}
