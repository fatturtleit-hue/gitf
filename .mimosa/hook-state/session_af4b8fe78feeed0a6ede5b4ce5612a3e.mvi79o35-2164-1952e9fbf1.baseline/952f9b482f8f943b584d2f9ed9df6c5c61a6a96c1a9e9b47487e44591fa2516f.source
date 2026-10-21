//go:build !windows

package handlers

import "syscall"

// getDiskStats returns (total, free, ok) for the given path.
func getDiskStats(path string) (total, free uint64, ok bool) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, 0, false
	}
	total = stat.Blocks * uint64(stat.Bsize)
	free = stat.Bavail * uint64(stat.Bsize)
	return total, free, true
}
