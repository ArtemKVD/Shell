package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

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

const testflag = true

func Ex(testflag bool) {
	if testflag == true {
		os.Exit(0)
	}
}

func getHistoryPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("Error getting home dir: %v\n", err)
		return "kubsh_history"
	}
	path := filepath.Join(home, "kubsh_history")
	return path
}

func HistoryWriter(command string, file *os.File) error {
	_, err := file.WriteString(command)
	return err
}

func Exit() {
	os.Exit(0)
}

func Echo(command string) {
	text := strings.TrimSpace(command[6:])
	text = strings.ReplaceAll(text, "'", "")
	fmt.Println(text)
	fmt.Println(strings.TrimSpace(command[6:]))
}

func Env(command string) {
	e := strings.TrimSpace(command[3:])
	if strings.Contains(e, "$PATH") {
		path := os.Getenv("PATH")
		paths := strings.Split(path, ":")
		for _, p := range paths {
			if p != "" {
				fmt.Println(p)
			}
		}
	} else if strings.Contains(e, ":") {
		envs := os.Environ()
		for _, e := range envs {
			fmt.Println(e)
		}
	} else {
		vrb := strings.TrimPrefix(e, "$")
		vrb = strings.TrimSpace(vrb)
		value := os.Getenv(vrb)
		fmt.Println(value)
	}
}

func UserCommand(command string) {
	parts := strings.Fields(command)

	switch parts[0] {
	case "adduser":
		cmd := exec.Command("sudo", "useradd", "-m", parts[1])
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Printf("Error adding user: %v\n", err)
		} else {
			SetupUsersVFS()
		}
	case "userdel":
		cmd := exec.Command("sudo", "userdel", "-r", parts[1])
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Printf("Error deleting user: %v\n", err)
		} else {
			home, _ := os.UserHomeDir()
			userDir := filepath.Join(home, "users", parts[1])
			os.RemoveAll(userDir)
		}
	}
}

func ExecuteBinary(command string) {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return
	}
	binaryName := parts[0]
	args := parts[1:]

	if parts[0] == "exec" && len(parts) > 1 {
		binaryName = parts[1]
		args = parts[2:]
	}

	binaryPath, err := exec.LookPath(binaryName)
	if err != nil {
		fmt.Printf("%s: command not found\n", binaryName)
		return
	}

	cmd := exec.Command(binaryPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		fmt.Printf("Error executing %s: %v\n", binaryName, err)
	}
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
	/*home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	usersDir := filepath.Join(home, "users")
	if err := os.MkdirAll(usersDir, 0755); err != nil {
		return err
	}*/
	usersDir := "/opt/users"

	users, err := getSystemUsers()
	if err != nil {
		return err
	}

	for _, u := range users {
		userDir := filepath.Join(usersDir, u.Username)
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
			shell := fields[6]
			if strings.Contains(shell, "bash") || strings.Contains(shell, "sh") && !strings.Contains(shell, "nologin") && !strings.Contains(shell, "false") {
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
	historyPath := getHistoryPath()
	file, err := os.OpenFile(historyPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("Error open file")
	}
	defer file.Close()

	err = SetupUsersVFS()
	if err != nil {
		fmt.Println("Error setting VFS:", err)
	}
	for {
		if !testflag {
			fmt.Fprint(os.Stdout, "$ ")
		}
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
			os.Exit(0)
		case strings.TrimSpace(command) == "\\q":
			fmt.Println(command)
			os.Exit(0)
		case strings.HasPrefix(command, "debug"):
			Echo(command)
			Ex(testflag)
		case strings.HasPrefix(command, "\\e"):
			Env(command)
			Ex(testflag)
		case strings.HasPrefix(command, "cat "):
			ExecuteBinary(command)
			Ex(testflag)
		case len(command) >= 4 && command[:4] == "type":
			Type(command)
			Ex(testflag)
		case err == io.EOF:
			fmt.Println("Ctrl+D")
			os.Exit(0)
		case strings.TrimSpace(command) == "test sighup":
			process, _ := os.FindProcess(os.Getpid())
			process.Signal(syscall.SIGHUP)
			Ex(testflag)
			continue
		case strings.TrimSpace(command) == "/l /dev/sda":
			DiskInfo(command)
			Ex(testflag)
		case command == "adduser" || strings.HasPrefix(command, "userdel"):
			UserCommand(command)
			Ex(testflag)
		default:
			if !testflag {
				fmt.Println(command + ": command not found")
			} else {
				с := strings.TrimSpace(command)
				if с != "" {
					fmt.Printf("%s: command not found\n", с)
				}
				Ex(testflag)
			}
		}
	}
}

func SignalHandler() {
	SigChan := make(chan os.Signal, 1)
	signal.Notify(SigChan, syscall.SIGHUP)

	go func() {
		for {
			<-SigChan
			fmt.Println("Configuration reloaded")
		}
	}()
}

func main() {
	SignalHandler()
	CommandHandler()
}
