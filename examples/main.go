package main

import (
	"fmt"

	gsmc "github.com/tsingroo/go-ssh-multi-command"
)

func main() {

	conn, err := gsmc.NewConnection("192.168.1.1:22", "abcd", "abcd")
	if err != nil {
		panic(err)
	}

	defer conn.Close()

	multiCmds := []gsmc.GsmcCommand{
		{
			CommandAndArgs: "su - root",
			ExpectRegExp:   ".*Password:$", // 正则含义：以任意字符开头，以Password:结尾
			UserInput:      "ansible",
		},
		{
			CommandAndArgs: "pwd",
		},
		{
			CommandAndArgs: "whoami",
		},
	}

	_, err = conn.ExecCommands(multiCmds)
	if err != nil {
		fmt.Println(err)
	}

}
