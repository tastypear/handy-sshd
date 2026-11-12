package cmd

import (
	"fmt"
	"github.com/nwtgck/handy-sshd"
	"github.com/nwtgck/handy-sshd/version"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
	"golang.org/x/exp/slog"
	"net"
	"os"
	"strconv"
	"strings"
	"bufio"
)

type flagType struct {
	//dnsServer    string
	showsVersion  bool
	sshHost       string
	sshPort       uint16
	sshUnixSocket string
	sshShell      string
	sshUsers      []string
	sshAuthKeys   []string

	allowTcpipForward       bool
	allowDirectTcpip        bool
	allowExecute            bool
	allowSftp               bool
	allowStreamlocalForward bool
	allowDirectStreamlocal  bool
}

type permissionFlagType = struct {
	name    string
	flagPtr *bool
}

type sshUser struct {
	name     string
	password string
}

func init() {
	cobra.OnInitialize()
}

func RootCmd() *cobra.Command {
	var flag flagType
	allPermissionFlags := []permissionFlagType{
		{name: "tcpip-forward", flagPtr: &flag.allowTcpipForward},
		{name: "direct-tcpip", flagPtr: &flag.allowDirectTcpip},
		{name: "execute", flagPtr: &flag.allowExecute},
		{name: "sftp", flagPtr: &flag.allowSftp},
		{name: "streamlocal-forward", flagPtr: &flag.allowStreamlocalForward},
		{name: "direct-streamlocal", flagPtr: &flag.allowDirectStreamlocal},
	}
	rootCmd := cobra.Command{
		Use:          os.Args[0],
		Short:        "handy-sshd",
		Long:         "Portable SSH server",
		SilenceUsage: true,
		Example: `# Listen on 2222 and accept user name "john" with password "mypass"
handy-sshd -u john:mypass

# Listen on 22 and accept the user without password
handy-sshd -p 22 -u john:

Permissions:
All permissions are allowed by default.
For example, specifying --allow-direct-tcpip and --allow-execute allows only them.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return rootRunEWithExtra(cmd, args, &flag, allPermissionFlags)
		},
	}

	rootCmd.PersistentFlags().BoolVarP(&flag.showsVersion, "version", "v", false, "show version")
	rootCmd.PersistentFlags().StringVarP(&flag.sshHost, "host", "", "", "SSH server host to listen (e.g. 127.0.0.1)")
	rootCmd.PersistentFlags().Uint16VarP(&flag.sshPort, "port", "p", 2222, "port to listen")
	// NOTE: long name 'unix-socket' is from curl (ref: https://curl.se/docs/manpage.html)
	rootCmd.PersistentFlags().StringVarP(&flag.sshUnixSocket, "unix-socket", "", "", "Unix domain socket to listen")
	rootCmd.PersistentFlags().StringVarP(&flag.sshShell, "shell", "", "", "Shell")
	//rootCmd.PersistentFlags().StringVar(&flag.dnsServer, "dns-server", "", "DNS server (e.g. 1.1.1.1:53)")
	rootCmd.PersistentFlags().StringArrayVarP(&flag.sshUsers, "user", "u", nil, `SSH user name (e.g. "john:mypass")`)
	rootCmd.PersistentFlags().StringArrayVarP(&flag.sshAuthKeys, "keys", "k", nil, "SSH authorized keys file. Can be used with --user so that both authentication methods are allowed")

	// Permission flags
	rootCmd.PersistentFlags().BoolVarP(&flag.allowTcpipForward, "allow-tcpip-forward", "", false, "client can use remote forwarding (ssh -R)")
	rootCmd.PersistentFlags().BoolVarP(&flag.allowDirectTcpip, "allow-direct-tcpip", "", false, "client can use local forwarding (ssh -L) and SOCKS proxy (ssh -D)")
	rootCmd.PersistentFlags().BoolVarP(&flag.allowExecute, "allow-execute", "", false, "client can use shell/interactive shell")
	rootCmd.PersistentFlags().BoolVarP(&flag.allowSftp, "allow-sftp", "", false, "client can use SFTP and SSHFS")
	rootCmd.PersistentFlags().BoolVarP(&flag.allowStreamlocalForward, "allow-streamlocal-forward", "", false, "client can use Unix domain socket remote forwarding (ssh -R)")
	rootCmd.PersistentFlags().BoolVarP(&flag.allowDirectStreamlocal, "allow-direct-streamlocal", "", false, "client can use Unix domain socket local forwarding (ssh -L)")

	return &rootCmd
}

func rootRunEWithExtra(cmd *cobra.Command, args []string, flag *flagType, allPermissionFlags []permissionFlagType) error {
	if flag.showsVersion {
		fmt.Fprintln(cmd.OutOrStdout(), version.Version)
		return nil
	}
	logger := slog.Default()

	// Allow all permissions if all permission is not set
	{
		allPermissionFalse := true
		for _, permissionFlag := range allPermissionFlags {
			allPermissionFalse = allPermissionFalse && !*permissionFlag.flagPtr
		}
		if allPermissionFalse {
			for _, permissionFlag := range allPermissionFlags {
				*permissionFlag.flagPtr = true
			}
		}
	}

	sshServer := &handy_sshd.Server{
		Logger:                  logger,
		AllowTcpipForward:       flag.allowTcpipForward,
		AllowDirectTcpip:        flag.allowDirectTcpip,
		AllowExecute:            flag.allowExecute,
		AllowSftp:               flag.allowSftp,
		AllowStreamlocalForward: flag.allowStreamlocalForward,
		AllowDirectStreamlocal:  flag.allowDirectStreamlocal,
	}
	var sshUsers []sshUser
	for _, u := range flag.sshUsers {
		splits := strings.SplitN(u, ":", 2)
		if len(splits) != 2 {
			return fmt.Errorf("invalid user format: %s", u)
		}
		sshUsers = append(sshUsers, sshUser{name: splits[0], password: splits[1]})
	}
	authorizedKeys := make(map[string]bool)
	for _, k := range flag.sshAuthKeys {
		hand, err := os.Open(k)
		if err != nil {
			logger.Warn(fmt.Sprintf("failed to open authorized keys file %q: %v", k, err))
			continue
		}
		scanner := bufio.NewScanner(hand)
		for scanner.Scan() {
			key := scanner.Text()
			pub, _, _, _, err := ssh.ParseAuthorizedKey([]byte(key))
			if err != nil { continue }
			authorizedKeys[string(ssh.FingerprintSHA256(pub))] = true
		}
	}
	// (base: https://gist.github.com/jpillora/b480fde82bff51a06238)
	//
	// handy-sshd: advertise an OpenSSH-shaped identification string. Clients key
	// wire-compat quirks off the server banner — e.g. ssh2 sends SSH_FXP_SYMLINK as
	// (targetpath, linkpath) only when the server identifies as OpenSSH, which is the
	// order pkg/sftp's server parses. Without this, symlink (and any other client-side
	// OpenSSH-compat path) breaks against handy-sshd while working against OpenSSH.
	// The SFTP subsystem's status codes, canonical status messages and errno mapping
	// are aligned with OpenSSH in the vendored pkg/sftp, so this is a compatibility
	// statement, not an impersonation.
	sshConfig := &ssh.ServerConfig{
		// OpenSSH-shaped banner — see the comment above.
		ServerVersion: "SSH-2.0-OpenSSH_10.0 (handy-sshd " + version.Version + "; OpenSSH-compatible)",
		PublicKeyCallback: func(metadata ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
            if authorizedKeys[string(ssh.FingerprintSHA256(key))] {
                return nil, nil
            }
            return nil, fmt.Errorf("public key rejected for %q", metadata.User())
        },
		//Define a function to run when a client attempts a password login
		PasswordCallback: func(metadata ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			for _, user := range sshUsers {
				// No auth required
				if user.name == metadata.User() && user.password == string(pass) {
					return nil, nil
				}
			}
			return nil, fmt.Errorf("password rejected for %q", metadata.User())
		},
		NoClientAuth: true,
		NoClientAuthCallback: func(metadata ssh.ConnMetadata) (*ssh.Permissions, error) {
			for _, user := range sshUsers {
				// No auth required
				if user.name == metadata.User() && user.password == "" {
					return nil, nil
				}
			}
			return nil, fmt.Errorf("%s auth required", metadata.User())
		},
	}
	// TODO: specify priv_key by flags
	pri, err := ssh.ParsePrivateKey([]byte(defaultHostKeyPem))
	if err != nil {
		return err
	}
	sshConfig.AddHostKey(pri)

	var ln net.Listener
	if flag.sshUnixSocket == "" {
		address := net.JoinHostPort(flag.sshHost, strconv.Itoa(int(flag.sshPort)))
		ln, err = net.Listen("tcp", address)
		if err != nil {
			return err
		}
		logger.Info(fmt.Sprintf("listening on %s...", address))
	} else {
		ln, err = net.Listen("unix", flag.sshUnixSocket)
		if err != nil {
			return err
		}
		logger.Info(fmt.Sprintf("listening on %s...", flag.sshUnixSocket))
	}
	defer ln.Close()

	showPermissions(logger, allPermissionFlags)

	for {
		conn, err := ln.Accept()
		if err != nil {
			logger.Error("failed to accept TCP connection", "err", err)
			continue
		}
		sshConn, chans, reqs, err := ssh.NewServerConn(conn, sshConfig)
		if err != nil {
			logger.Info("failed to handshake", "err", err)
			conn.Close()
			continue
		}
		logger.Info("new SSH connection", "remote_address", sshConn.RemoteAddr(), "client_version", string(sshConn.ClientVersion()))
		go sshServer.HandleGlobalRequests(sshConn, reqs)
		go sshServer.HandleChannels(flag.sshShell, chans)
	}
}

func showPermissions(logger *slog.Logger, allPermissionFlags []permissionFlagType) {
	var allowedList []string
	var notAllowedList []string
	for _, permissionFlag := range allPermissionFlags {
		if *permissionFlag.flagPtr {
			allowedList = append(allowedList, `"`+permissionFlag.name+`"`)
		} else {
			notAllowedList = append(notAllowedList, `"`+permissionFlag.name+`"`)
		}
	}
	showList := func(l []string) string {
		if len(l) == 0 {
			return "none"
		}
		return strings.Join(l, ", ")
	}
	logger.Info(fmt.Sprintf("allowed: %s", showList(allowedList)))
	logger.Info(fmt.Sprintf("NOT allowed: %s", showList(notAllowedList)))
}
