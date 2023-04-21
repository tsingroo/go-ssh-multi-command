package gsmc

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"regexp"
	"time"

	"golang.org/x/crypto/ssh"
)

type GsmcConnection struct {
	*ssh.Client
	password string
}

// GsmcCommand
type GsmcCommand struct {
	CommandAndArgs string
	ExpectRegExp   string
	UserInput      string
	TimeoutSeconds int
}

func NewConnection(addr, user, password string) (*GsmcConnection, error) {
	sshConfig := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.HostKeyCallback(func(hostname string, remote net.Addr, key ssh.PublicKey) error { return nil }),
	}

	conn, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return nil, err
	}

	return &GsmcConnection{conn, password}, nil

}

func (conn *GsmcConnection) ExecCommands(cmds []GsmcCommand) ([]byte, error) {
	cmdCount := len(cmds)
	gIndex := 0

	session, err := conn.NewSession()
	if err != nil {
		log.Fatal(err)
	}
	defer session.Close()

	modes := ssh.TerminalModes{
		ssh.ECHO:          0,     // disable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}

	err = session.RequestPty("xterm", 80, 40, modes)
	if err != nil {
		return []byte{}, err
	}

	in, err := session.StdinPipe()
	if err != nil {
		log.Fatal(err)
	}

	out, err := session.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}

	var output []byte

	go func(in io.WriteCloser, out io.Reader, output *[]byte) {
		var (
			line string
			r    = bufio.NewReader(out)
		)
		for {
			b, err := r.ReadByte()
			if err != nil {
				fmt.Println("err:", err)
				break
			}

			*output = append(*output, b)

			if b == byte('\n') {
				fmt.Println(line)
				line = ""
				continue
			}

			line += string(b)

			regPattern := cmds[gIndex].ExpectRegExp
			if regPattern != "" {
				regExp := regexp.MustCompile(regPattern)
				if regExp.MatchString(line) {
					_, err = in.Write([]byte(cmds[gIndex].UserInput + "\n"))
					if err != nil {
						break
					}
				}
			}

		}
	}(in, out, &output)

	err = session.Shell()
	if err != nil {
		return []byte{}, err
	}

	for gIndex = 0; gIndex < cmdCount; gIndex++ {
		in.Write([]byte(cmds[gIndex].CommandAndArgs + "\n"))
		time.Sleep(time.Duration(cmds[gIndex].TimeoutSeconds) * time.Second)
	}

	return output, nil
}
