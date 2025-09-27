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

func HistoryWriter(command string, file *os.File) error {
	_, err := file.WriteString(command)
	return err
}
func Exit() {
	os.Exit(0)
}

func Echo(command string) {
	fmt.Println(command[4:])
}

func Env(command string) error {
	err := godotenv.Load()
	if strings.Contains(command, ":") {
		envs := os.Environ()
		for _, e := range envs {
			fmt.Println(e)
		}
	} else {
		path := strings.TrimSpace(command[4:])
		fmt.Println(os.Getenv(path))
	}
	return err
}

func Type(command string) {
	switch true {
	case strings.Contains(command, "echo"):
		fmt.Println("echo is a shell builtin")
	case strings.Contains(command, "exit 0"):
		fmt.Println("exit is a shell builtin")
	case strings.Contains(command, "/q"):
		fmt.Println("/e is a shell builtin")
	case strings.Contains(command, "/e $"):
		fmt.Println("/e $ is a shell builtin")
	}
}

func CommandHandler() {
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
		err = HistoryWriter(command, file)
		if err != nil {
			fmt.Println("History write error")
		}
		switch true {
		case strings.TrimSpace(command) == "exit 0":
			Exit()
		case strings.TrimSpace(command) == "/q":
			fmt.Println(command)
			Exit()
		case strings.Contains(command, "echo"):
			Echo(command)
		case strings.Contains(command, "/e $"):
			err := Env(command)
			if err != nil {
				fmt.Println("Godotenv load error")
			}
		case len(command) >= 4 && command[:4] == "type":
			Type(command)
		case err == io.EOF:
			fmt.Println("Ctrl+D")
			os.Exit(0)
		case strings.TrimSpace(command) == "test sighup":
			process, _ := os.FindProcess(os.Getpid())
			process.Signal(syscall.SIGHUP)
			continue
		default:
			fmt.Println(command[:len(command)-1] + ": command not found")
		}
	}
}

func SignalHandler() {
	SigChan := make(chan os.Signal, 1)
	signal.Notify(SigChan, syscall.SIGHUP)

	go func() {
		for {
			<-SigChan
			fmt.Println("Configuration reload")
		}
	}()
}

func main() {
	SignalHandler()
	CommandHandler()
}
