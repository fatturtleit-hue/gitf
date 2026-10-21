//go:build windows

package handlers

// getDiskStats returns (total, free, ok) for the given path on Windows.
// Uses GetDiskFreeSpaceExW via golang.org/x/sys/windows — but to avoid
// adding a dependency, simply report disk stats as unavailable on Windows.
func getDiskStats(path string) (total, free uint64, ok bool) {
	return 0, 0, false
}
