package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/dagger-project/daginit/internal/pkg/configuration"
)

const TMP_PATH = "/tmp/daginit"
const RELROOT = "/home/kevsmith/repos/dagger_project/demo/_build/prod/rel/demo"
const COOKIE = "3WIIBVLHE75VUAU65QQLHFGUTMX3LAJQEMUNIVKZM7436D3AIMAA===="

func main() {
	notifier := make(chan os.Signal, 1)
	signal.Notify(notifier, syscall.SIGTERM, syscall.SIGCHLD)
	os.Clearenv()
	config, err := configuration.Load()
	if err != nil {
		panic(err)
	}
	config.Logger.Info("daginit configuration loaded")
	var out, errOut *os.File = nil, nil
	if config.SaveStdOutErr {
		os.MkdirAll(TMP_PATH, 0777)
		out, err = os.CreateTemp(TMP_PATH, "out_*")
		if err != nil {
			config.Logger.Panic("Error opening redirected output stream: %v", err)
		}
		errOut, err = os.CreateTemp(TMP_PATH, "err_*")
		if err != nil {
			config.Logger.Panic("Error opening redirected error stream: %v", err)
		}
	}
	config.Logger.Info("I/O redirection set up")
	bootScript := config.MakeReleasePath("releases/%s/elixir")
	config.Logger.Info("Booting release using %s", bootScript)
	args := []string{
		"",
		"--cookie",
		COOKIE,
		"--boot",
		config.MakeReleasePath("releases/%s/start"),
		"--boot-var",
		"RELEASE_LIB",
		config.MakeReleasePath("lib"),
		"--sname",
		"demo",
		"--erl",
		"+fnue -mode embedded",
		"--erl-config",
		config.MakeReleasePath("releases/%s/sys"),
		"--vm-args",
		config.MakeReleasePath("releases/%s/vm.args"),
		"--no-halt",
	}
	devNullFd := uintptr(0)
	devNull, err := os.OpenFile(os.DevNull, os.O_RDWR, 0644)
	if err == nil {
		devNullFd = devNull.Fd()
	}
	redirectedFiles := []uintptr{devNullFd, devNullFd, devNullFd}
	if out != nil {
		redirectedFiles[1] = out.Fd()
	}
	if errOut != nil {
		redirectedFiles[2] = errOut.Fd()
	}
	attr := syscall.ProcAttr{
		Files: redirectedFiles,
		Env:   []string{"HOME=/home/kevsmith"},
	}
	pid, err := syscall.ForkExec(bootScript, args, &attr)
	if err != nil {
		config.Logger.Panic("Error starting release: %v", err)
	} else {
		config.Logger.Info("Release running on OS process %d", pid)
	}
	for {
		s := <-notifier
		fmt.Printf("Got signal: %+v\n", s)
		if errOut != nil {
			err = errOut.Close()
			config.Logger.Error("Error closing error stream: %v", err)
		}
		if err != nil {
			err = out.Close()
			if err != nil {
				config.Logger.Error("Error closing output stream: %v", err)
			}
		}
		os.Exit(0)
	}
}
