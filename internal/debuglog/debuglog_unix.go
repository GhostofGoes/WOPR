//go:build unix

package debuglog

import (
	"fmt"
	"os"
	"syscall"
)

const oNoFollow = syscall.O_NOFOLLOW

// checkDir accepts the log's directory only if it is a real directory (not a symbolic link)
// that this user owns, and takes away any access it gives to other users.
func checkDir(d string) error {
	info, err := os.Lstat(d)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", d)
	}
	if st, ok := info.Sys().(*syscall.Stat_t); ok && int(st.Uid) != os.Getuid() {
		return fmt.Errorf("%s belongs to another user", d)
	}
	if info.Mode().Perm()&0o077 != 0 {
		return os.Chmod(d, 0o700)
	}
	return nil
}
