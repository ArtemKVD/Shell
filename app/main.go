package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

// Ensures gofmt doesn't remove the "fmt" import in stage 1 (feel free to remove this!)
var _ = fmt.Fprint

func main() {
	file, err := os.OpenFile("../kubsh_history.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error open file")
	}
	defer file.Close()

	for {
		fmt.Fprint(os.Stdout, "$ ")
		command, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil && err != io.EOF {
			fmt.Println("Reader error")
		}
		_, errf := file.WriteString(command)
		if errf != nil {
			fmt.Println("History write error")
		}

		if strings.TrimSpace(command) == "exit 0" {
			fmt.Println(command)
			os.Exit(0)
		}
		if strings.TrimSpace(command) == "/q" {
			fmt.Println(command)
			os.Exit(0)
		}
		if strings.Contains(command, "echo") {
			fmt.Println(command[4:])
		}

		if len(command) >= 4 && command[:4] == "type" {
			if strings.Contains(command, "echo") {
				fmt.Println("echo is a shell builtin")
			}
			if strings.Contains(command, "exit") {
				fmt.Println("exit is a shell builtin")
			}
		}
		if err == io.EOF {
			fmt.Println("Ctrl+D")
			os.Exit(0)
		}
		//fmt.Println(command[:len(command)-1] + ": command not found")
	}
}
