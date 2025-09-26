package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/joho/godotenv"
)

// Ensures gofmt doesn't remove the "fmt" import in stage 1 (feel free to remove this!)
var _ = fmt.Fprint

func main() {
	file, err := os.OpenFile("../kubsh_history.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error open file")
	}
	defer file.Close()

	SigChan := make(chan os.Signal, 1)
	signal.Notify(SigChan, syscall.SIGHUP)

	go func() {
		for {
			<-SigChan
			fmt.Println("Configuration reload")
		}
	}()

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

		if strings.TrimSpace(command) == "test sighup" {
			process, _ := os.FindProcess(os.Getpid())
			process.Signal(syscall.SIGHUP)
			continue
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

		if strings.Contains(command, "/e $") {
			err := godotenv.Load()
			if err != nil {
				fmt.Println("godotenv load error")
			}
			if strings.Contains(command, ":") {
				envs := os.Environ()
				for _, e := range envs {
					fmt.Println(e)
				}
			} else {
				path := strings.TrimSpace(command[4:])
				fmt.Println(os.Getenv(path))
			}
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
