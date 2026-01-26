package main

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

const (
	DirMode           = 0755
	FileMode          = 0644
	cgroupProcsFile   = "cgroup.procs"
	processCGroupFile = "/proc/self/cgroup"
	baseCGroupPath    = "/sys/fs/cgroup"
)

var (
	childProcessCGroup = createNewCGroup("deckmaster.scope")
)

func runningCGroup() string {
	s, err := os.ReadFile(processCGroupFile)
	if err != nil {
		errorLogF("Unable to read the control group for the current process")
		panic(err)
	}
	split := strings.Split(string(s), ":")
	cgroup := split[len(split)-1]
	return strings.TrimSpace(cgroup)
}

func createNewCGroup(name string) string {
	cgroupParent := filepath.Dir(runningCGroup())
	cgroupPath := path.Join(baseCGroupPath, cgroupParent, name)
	if err := os.MkdirAll(cgroupPath, DirMode); err != nil {
		errorLogF("Unable to create new control group for child processes\n\t", cgroupPath)
		panic(err)
	}
	verboseLog("Using control group to spawn child processes\n\t%s", cgroupPath)
	return cgroupPath
}

func moveProcessToCGroup(pid int, cgroup string) error {
	cgroupFile := path.Join(cgroup, cgroupProcsFile)
	fileContents := fmt.Sprintf("%d\n", pid)
	return os.WriteFile(cgroupFile, []byte(fileContents), FileMode)
}

func expandExecutable(exe string) string {
	for _, base := range PATH {
		cmd := filepath.Join(base, exe)
		s, e := os.Stat(cmd)
		if e != nil || s.IsDir() {
			continue
		}
		fileMode := s.Mode()
		if fileMode&0111 != 0 {
			return cmd
		}
	}
	return exe
}

// executes a command.
func executeCommand(cmd string) error {
	args := SPACES.Split(cmd, -1)
	exe := expandExecutable(args[0])

	command := exec.Command(exe, args[1:]...)
	if err := command.Start(); err != nil {
		errorLogF("failed to execute '%s %s'", exe, args[1:])
		return err
	}
	pid := command.Process.Pid
	if err := moveProcessToCGroup(pid, childProcessCGroup); err != nil {
		errorLog(err, "Unable to move child process %d to cgroup", pid)
	}
	return command.Process.Release()
}
