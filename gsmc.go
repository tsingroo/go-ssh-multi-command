package gsmc

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"regexp"
	"strings"

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

	err = session.Shell()
	if err != nil {
		return []byte{}, err
	}

	clearLoginMsg(in, out, &output)

	for cmdIndex := 0; cmdIndex < cmdCount; cmdIndex++ {
		cmdStr := cmds[cmdIndex].CommandAndArgs
		if !strings.Contains(cmdStr, "su -") {
			cmdStr = cmdStr + ";echo 'exit code:'$?\n"
		} else {
			cmdStr = cmdStr + "\n"
		}
		in.Write([]byte(cmdStr))
		fmt.Print(cmdStr)
		cmds[cmdIndex].CommandAndArgs = cmdStr
		err := HandleStdInStdoutStdErr(in, out, &output, cmds[cmdIndex])
		if err != nil {
			return []byte{}, err
		}
	}

	return output, nil
}

func HandleStdInStdoutStdErr(stdIn io.WriteCloser, stdOut io.Reader, output *[]byte, cmd GsmcCommand) error {
	var (
		line  string
		r           = bufio.NewReader(stdOut)
		rtErr error = nil
	)
	for {
		b, err := r.ReadByte()
		if err != nil {
			if err != io.EOF {
				rtErr = err
			}
			break
		}

		*output = append(*output, b)

		if b == byte('\n') {
			fmt.Println(line)
			line = ""
			continue
		}

		line += string(b)
		if isCmdComplete(line) {
			fmt.Print(line)
			break
		}

		if strings.Contains(line, "Authentication failure") {
			rtErr = errors.New("authentication failure")
			break
		}
		exitCodeRegParrern := regexp.MustCompile("exit code:[0-9]+")
		if exitCodeRegParrern.MatchString(line) {
			if !strings.Contains(line, "exit code:0") {
				rtErr = errors.New(cmd.CommandAndArgs + " 命令执行出错")
				break
			} else {
				break
			}
		}

		regPattern := cmd.ExpectRegExp
		if regPattern != "" {
			regExp := regexp.MustCompile(regPattern)
			if regExp.MatchString(line) {
				_, err = stdIn.Write([]byte(cmd.UserInput + "\n"))
				fmt.Print(cmd.UserInput + "\n")
				if err != nil {
					rtErr = err
					break
				}
			}
		}
	}

	return rtErr
}

// clearLoginMsg 清除ssh初次登录产生的信息
func clearLoginMsg(stdIn io.WriteCloser, stdOut io.Reader, output *[]byte) {
	HandleStdInStdoutStdErr(stdIn, stdOut, output, GsmcCommand{})
}

// isCmdComplete 判断命令是否执行完毕
func isCmdComplete(line string) bool {
	// 判断条件：是否重新出现命令提示符。root用户的命令提示符为#结尾，普通用户的命令提示符为$结尾
	if strings.HasSuffix(line, "~#") {
		return true
	}
	if strings.HasSuffix(line, "~]#") {
		return true
	}
	if strings.HasSuffix(line, "~]$") {
		return true
	}
	if line == "$" {
		return true
	}
	return false
}
