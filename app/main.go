package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/shirou/gopsutil/disk"
)

type UserInfo struct {
	Username string
	Uid      string
	Gid      string
	Name     string
	HomeDir  string
	Shell    string
}

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

func DiskInfo(command string) {
	parts := strings.Fields(command)
	device := "/dev/sda"
	if len(parts) > 1 {
		device = parts[1]
	}

	partitions, err := disk.Partitions(false)
	if err != nil {
		fmt.Println("Error getting disk info: ", err)
		return
	}

	tf := false
	for _, partition := range partitions {
		if partition.Device == device || strings.HasPrefix(partition.Device, device) {
			if strings.Contains(partition.Mountpoint, "/boot") || strings.Contains(partition.Mountpoint, "/efi") || strings.Contains(partition.Mountpoint, "/EFI") {
				tf = true
				break
			}
		}
		if partition.Fstype == "vfat" || partition.Fstype == "efi" {
			tf = true
			break
		}
	}

	if tf {
		fmt.Println("Disk loaded")
	} else {
		fmt.Println("disk not loaded")
	}
}

func SetupUsersVFS() error {
	if err := os.MkdirAll("Users", 0755); err != nil {
		return err
	}

	users, err := getSystemUsers()
	if err != nil {
		return err
	}

	for _, u := range users {
		userDir := filepath.Join("Users", u.Username)
		if err := os.MkdirAll(userDir, 0755); err != nil {
			continue
		}

		if err := os.WriteFile(filepath.Join(userDir, "id"),
			[]byte(u.Uid), 0644); err != nil {
			continue
		}

		if err := os.WriteFile(filepath.Join(userDir, "home"),
			[]byte(u.HomeDir), 0644); err != nil {
			continue
		}

		if err := os.WriteFile(filepath.Join(userDir, "shell"),
			[]byte(u.Shell), 0644); err != nil {
			continue
		}
	}
	return nil
}

func getSystemUsers() ([]UserInfo, error) {
	var users []UserInfo
	file, err := os.Open("/etc/passwd")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Split(line, ":")
		if len(fields) >= 7 {
			uid := fields[2]
			if uidInt, err := strconv.Atoi(uid); err == nil && uidInt >= 1000 {
				users = append(users, UserInfo{
					Username: fields[0],
					Uid:      uid,
					Gid:      fields[3],
					Name:     fields[4],
					HomeDir:  fields[5],
					Shell:    fields[6],
				})
			}
		}
	}

	return users, scanner.Err()
}

func CommandHandler() {
	file, err := os.OpenFile("../kubsh_history.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error open file")
	}
	defer file.Close()

	err = SetupUsersVFS()
	if err != nil {
		fmt.Println("Error setting VFS:", err)
	}
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
		case strings.TrimSpace(command) == "/l /dev/sda":
			DiskInfo(command)
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
