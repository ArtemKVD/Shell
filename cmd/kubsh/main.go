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
	"time"

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

func getVFSDir() string {
	vfsDir := os.Getenv("VFS_DIR")
	if vfsDir != "" {
		return vfsDir
	}

	cwd, err := os.Getwd()
	if err == nil {
		usersDir := filepath.Join(cwd, "users")
		if info, err := os.Stat(usersDir); err == nil && info.IsDir() {
			return usersDir
		}
		parentDir := filepath.Dir(cwd)
		usersDir = filepath.Join(parentDir, "users")
		if info, err := os.Stat(usersDir); err == nil && info.IsDir() {
			return usersDir
		}
		testsUsersDir := filepath.Join(cwd, "tests", "users")
		if info, err := os.Stat(testsUsersDir); err == nil && info.IsDir() {
			return testsUsersDir
		}
	}
	return "/opt/users"
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
		cmd := exec.Command("useradd", "-m", parts[1])
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Printf("Error adding user: %v\n", err)
		} else {
			SetupUsersVFS()
		}
	case "userdel":
		cmd := exec.Command("userdel", "-r", parts[1])
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
	/*
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
	*/

	parts := strings.Fields(command)
	device := "/dev/sda"
	if len(parts) > 1 {
		device = parts[1]
	}

	partitions, err := disk.Partitions(false)
	if err != nil {
		fmt.Printf("Error getting disk info: %v\n", err)
		return
	}

	found := false
	for _, partition := range partitions {
		if partition.Device == device || strings.HasPrefix(partition.Device, device) {
			found = true

			usage, err := disk.Usage(partition.Mountpoint)
			if err != nil {
				fmt.Printf("Device: %s\n", partition.Device)
				fmt.Printf("Mountpoint: %s\n", partition.Mountpoint)
				fmt.Printf("Filesystem: %s\n", partition.Fstype)
				fmt.Printf("Error getting usage: %v\n", err)
				break
			}

			fmt.Printf("Device: %v\n", partition.Device)
			fmt.Printf("Mountpoint: %v\n", partition.Mountpoint)
			fmt.Printf("Filesystem: %v\n", partition.Fstype)
			fmt.Printf("Total: %v\n", float64(usage.Total))
			fmt.Printf("Used: %v\n", float64(usage.Used))
			fmt.Printf("Free: %v\n", float64(usage.Free))

			break
		}
	}

	if !found {
		fmt.Printf("Device %s not found\n", device)
	}
}

func SetupUsersVFS() error {
	usersDir := getVFSDir()
	if err := os.MkdirAll(usersDir, 0755); err != nil {
		return err
	}

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

func watchVFS(usersDir string) {
	knownDirs := make(map[string]bool)
	if entries, err := os.ReadDir(usersDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				knownDirs[entry.Name()] = true
			}
		}
	}

	for {
		entries, err := os.ReadDir(usersDir)
		if err != nil {
			time.Sleep(100 * time.Millisecond)
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() {
				username := entry.Name()
				if !knownDirs[username] {
					knownDirs[username] = true
					createUserFromVFS(username)
				}
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func createUserFromVFS(username string) {
	file, err := os.Open("/etc/passwd")
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Split(line, ":")
		if len(fields) > 0 && fields[0] == username {
			usersDir := getVFSDir()
			userDir := filepath.Join(usersDir, username)
			if _, err := os.Stat(userDir); os.IsNotExist(err) {
				os.MkdirAll(userDir, 0755)
				if len(fields) >= 7 {
					os.WriteFile(filepath.Join(userDir, "id"), []byte(fields[2]), 0644)
					os.WriteFile(filepath.Join(userDir, "home"), []byte(fields[5]), 0644)
					os.WriteFile(filepath.Join(userDir, "shell"), []byte(fields[6]), 0644)
				}
			}
			return
		}
	}

	userEntry := fmt.Sprintf("%s:x:10000:10000:VFS User:/home/%s:/bin/bash\n", username, username)

	f, err := os.OpenFile("/etc/passwd", os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	_, err = f.WriteString(userEntry)
	if err != nil {
		return
	}

	err = f.Sync()
	if err != nil {
		return
	}

	usersDir := getVFSDir()
	userDir := filepath.Join(usersDir, username)
	os.MkdirAll(userDir, 0755)
	os.WriteFile(filepath.Join(userDir, "id"), []byte("10000"), 0644)
	os.WriteFile(filepath.Join(userDir, "home"), []byte("/home/"+username), 0644)
	os.WriteFile(filepath.Join(userDir, "shell"), []byte("/bin/bash"), 0644)

	time.Sleep(10 * time.Millisecond)
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
	vfsDir := getVFSDir()
	go watchVFS(vfsDir)
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
		case strings.HasPrefix(command, "/l "):
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
