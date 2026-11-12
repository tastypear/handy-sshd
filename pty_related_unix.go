//go:build !windows
// +build !windows

// NOTE: pty.Start() is not supported in Windows

package handy_sshd

import (
	"github.com/creack/pty"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh"
	"io"
	"os"
	"os/exec"
	"sync"
)

func (s *Server) createPty(shell, term string, connection ssh.Channel) (*os.File, error) {
	// handy-sshd: OpenSSH parity — run the LOGIN USER's shell (flag > $SHELL >
	// /etc/passwd > sh) instead of the raw process env. Previously a process env
	// without $SHELL (e.g. minimal containers) silently landed interactive sessions
	// in "sh" (dash), which has no line editing — arrow/control keys print their
	// escape/control sequences literally.
	shell = resolveShell(shell)
	sh := exec.Command(shell)

	// OpenSSH parity: TERM comes from the pty-req payload, SHELL reflects the session
	// shell (ttyd behaves the same way). Appended last so they override any inherited
	// values (Go's os/exec keeps the last occurrence of a duplicate key).
	env := append(os.Environ(),
		"SHELL="+shell,
	)
	if term != "" {
		env = append(env, "TERM="+term)
	}
	sh.Env = env

	// Prepare teardown function
	closer := func() {
		connection.SendRequest("exit-status", false, ssh.Marshal(exitStatusMsg{
			Status: 0,
		}))
		connection.Close()
		if sh.Process != nil {
			_, err := sh.Process.Wait()
			if err != nil {
				s.Logger.Info("failed to exit shell", err)
			}
		}
		s.Logger.Info("session closed")
	}

	// Allocate a terminal for this channel
	s.Logger.Info("creating pty...")
	shf, err := pty.Start(sh)
	if err != nil {
		s.Logger.Info("failed to start pty", "err", err)
		closer()
		return nil, errors.Errorf("could not start pty (%s)", err)
	}

	// pipe session to bash and visa-versa
	var once sync.Once
	go func() {
		io.Copy(connection, shf)
		once.Do(closer)
	}()
	go func() {
		io.Copy(shf, connection)
		once.Do(closer)
	}()
	return shf, nil
}

// setWinsize sets the size of the given pty.
func setWinsize(t *os.File, w, h uint32) error {
	return pty.Setsize(t, &pty.Winsize{Rows: uint16(h), Cols: uint16(w)})
}
