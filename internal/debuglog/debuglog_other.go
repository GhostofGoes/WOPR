//go:build !unix

package debuglog

import (
	"fmt"
	"os"
)

const oNoFollow = 0 // no O_NOFOLLOW on Windows; Open's Lstat refuses a link that is already there

// checkDir accepts the log's directory only if it is a real directory, not a link to one.
// Windows keeps the user's cache directory private by its ACLs.
func checkDir(d string) error {
	info, err := os.Lstat(d)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", d)
	}
	return nil
}
