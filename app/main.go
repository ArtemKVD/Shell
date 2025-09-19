package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
)

// Ensures gofmt doesn't remove the "fmt" import in stage 1 (feel free to remove this!)
var _ = fmt.Fprint

func main() {
	for {
		fmt.Fprint(os.Stdout, "$ ")
		command, err := bufio.NewReader(os.Stdin).ReadString('\n')
		if err != nil {
			log.Println("Reader error")
		}
		if strings.TrimSpace(command) == "exit 0" {
			fmt.Println(command)
			os.Exit(0)
		}
		if strings.Contains(command, "echo") {
			fmt.Println(command[4:])
		}

		if command[:4] == "type" {
			if strings.Contains(command, "echo") {
				fmt.Printf("echo is a shell builtin")
			}
			if strings.Contains(command, "exit") {
				fmt.Printf("exit is a shell builtin")
			}

		}
		//fmt.Println(command[:len(command)-1] + ": command not found")
	}
}
