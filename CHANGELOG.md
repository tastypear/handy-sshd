# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](http://keepachangelog.com/en/1.0.0/)

## [Unreleased]

## [0.6.1] - 2026-11-13
### Fixed
* exec requests were never answered with CHANNEL_SUCCESS/FAILURE since the 0.6.0
  refactor (the Reply call was dropped when the pipe wiring moved into
  runShellProcess) — npm ssh2 clients treated every exec as failed ("Unable to
  exec") even though the command ran, breaking agent-shim's exec bridge. Now:
  Reply(true) on accept, Reply(false) on payload parse failure, and
  exit-status(127) + close if pipe setup fails.

## [0.6.0] - 2026-11-12
### Changed
* SFTP responses aligned with OpenSSH: errno table (ENOTDIR/ELOOP/EBADF -> NO_SUCH_FILE,
  EINVAL/ENAMETOOLONG -> BAD_MESSAGE, ENOSYS -> OP_UNSUPPORTED), canonical openssh status
  messages ("No such file", "Failure", ...) instead of Go error texts, and an
  OpenSSH-shaped server banner so clients apply their OpenSSH compatibility paths
  (fixes e.g. ssh2 symlink ordering)
* Interactive PTY sessions now run the login user's shell (resolveShell) with TERM from
  pty-req and SHELL in the session env — no more silent fallback to dash when the
  launcher env lacks $SHELL

### Fixed
* A bare "shell" session request (no pty) is accepted but never started a process —
  the channel hung forever; it now runs the login shell over plain pipes (ssh -T
  behavior) and reports the exit status

## [0.4.3] - 2024-05-27
### Changed
* Update dependencies

### Fixed
* Wait non-nil process when pty open failed

## [0.4.2] - 2024-04-05
### Changed
* Update dependencies

## [0.4.1] - 2023-09-13
### Changed
* Update dependencies

## [0.4.0] - 2023-08-11
### Added
* Add `-u` short flag for `--user`
* Support "cancel-tcpip-forward" and "cancel-streamlocal-forward@openssh.com"

### Changed
* Add more examples to help message

## [0.3.0] - 2023-08-11
### Added
* Support Unix domain socket local port forwarding
* Support Unix domain socket remote port forwarding

### Fixed
* Handle multiple global requests simultaneously

## [0.2.1] - 2023-08-09
### Changed
* Update dependencies

## [0.2.0] - 2023-08-09
### Added
* Add permissions

### Changed
* Require one user at least

## 0.1.0 - 2023-08-08
### Added
* Initial release

[Unreleased]: https://github.com/nwtgck/handy-sshd/compare/v0.4.2...HEAD
[0.4.3]: https://github.com/nwtgck/handy-sshd/compare/v0.4.2...v0.4.3
[0.4.2]: https://github.com/nwtgck/handy-sshd/compare/v0.4.1...v0.4.2
[0.4.1]: https://github.com/nwtgck/handy-sshd/compare/v0.4.0...v0.4.1
[0.4.0]: https://github.com/nwtgck/handy-sshd/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/nwtgck/handy-sshd/compare/v0.2.1...v0.3.0
[0.2.1]: https://github.com/nwtgck/handy-sshd/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/nwtgck/handy-sshd/compare/v0.1.0...v0.2.0
