//go:build !plan9
// +build !plan9

package sftp

import (
	"os"
	"syscall"
)

const EBADF = syscall.EBADF

func wrapPathError(filepath string, err error) error {
	if errno, ok := err.(syscall.Errno); ok {
		return &os.PathError{Path: filepath, Err: errno}
	}
	return err
}

// translateErrno translates a syscall error number to a SFTP error code.
//
// handy-sshd: OpenSSH parity patch — table mirrors errno_to_portable() in
// openssh sftp-server.c so status codes match OpenSSH servers exactly
// (notably ENOTDIR/ELOOP/EBADF -> NO_SUCH_FILE, where upstream pkg/sftp
// returned FAILURE).
func translateErrno(errno syscall.Errno) uint32 {
	switch errno {
	case 0:
		return sshFxOk
	case syscall.ENOENT, syscall.ENOTDIR, syscall.EBADF, syscall.ELOOP:
		return sshFxNoSuchFile
	case syscall.EPERM, syscall.EACCES, syscall.EFAULT:
		return sshFxPermissionDenied
	case syscall.ENAMETOOLONG, syscall.EINVAL:
		return sshFxBadMessage
	case syscall.ENOSYS:
		return sshFxOPUnsupported
	}

	return sshFxFailure
}

func translateSyscallError(err error) (uint32, bool) {
	switch e := err.(type) {
	case syscall.Errno:
		return translateErrno(e), true
	case *os.PathError:
		debug("statusFromError,pathError: error is %T %#v", e.Err, e.Err)
		if errno, ok := e.Err.(syscall.Errno); ok {
			return translateErrno(errno), true
		}
	}
	return 0, false
}
