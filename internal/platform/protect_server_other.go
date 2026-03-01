//go:build !linux

package platform

import "io"

func ServeProtect(path string, verbose bool, fwmark int, protectCtl func(fd int)) io.Closer {
	return nil
}
