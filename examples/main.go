package main

import gsmc "github.com/tsingroo/go-ssh-multi-command"

func main() {
	multiCmds := []gsmc.GsmcCommand{
		{
			CommandAndArgs: "su - root",
			ExpectRegExp:   ".*Password:$", // 正则含义：以任意字符开头，以Password:结尾
			UserInput:      "ansible",
			TimeoutSeconds: 1,
		},
		{
			CommandAndArgs: "pwd",
			ExpectRegExp:   "",
			UserInput:      "",
			TimeoutSeconds: 1,
		},
		{
			CommandAndArgs: "which nginx",
			ExpectRegExp:   "",
			UserInput:      "",
			TimeoutSeconds: 1,
		},
	}
	conn, err := gsmc.NewConnection("192.168.2.244:22", "gmsnmp", "gmsnmp")
	if err != nil {
		panic(err)
	}

	defer conn.Close()

	_, err = conn.ExecCommands(multiCmds)
	if err != nil {
		panic(err)
	}

}
