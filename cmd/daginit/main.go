package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"syscall"

	"github.com/dagger-project/daginit/internal/pkg/configuration"
)

const TMP_PATH = "/tmp/daginit"

func sanitizedCommand(command string) string {
	re := regexp.MustCompile(`\-\-cookie ([a-z]|[A-Z]|[0-9]|\-|_|=)+`)
	return re.ReplaceAllString(command, "--cookie **********")
}

func main() {
	os.Clearenv()
	configFile := ""
	if len(os.Args) > 1 {
		configFile = os.Args[1]
	}
	config, err := configuration.Load(configFile)
	if err != nil {
		panic(err)
	}
	config.Logger.Info("daginit configuration loaded")
	bootScript := config.MakeReleasePath("releases/%s/elixir")
	config.Logger.Info("Booting release using %s", bootScript)
	cmd := exec.Command(bootScript)
	cmd.Args = []string{
		"elixir",
		"--cookie",
		config.Cookie,
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
	if config.LogStdOut || config.LogStdErr {
		outputLogPath := config.MakeReleasePath("releases/%s/logs")
		err = os.MkdirAll(outputLogPath, 0777)
		if err != nil {
			config.Logger.Panic("Error setting up I/O redirection: %v", err)
		}
	}
	if config.LogStdOut {
		outPath := config.MakeReleasePath("releases/%s/logs/stdout.log")
		out, err := os.Create(outPath)
		if err != nil {
			config.Logger.Panic("Error setting up stdout: %v", err)
		}
		cmd.Stdout = out
		config.Logger.Info("stdout logged to %s", outPath)
	} else {
		config.Logger.Info("stdout logged to /dev/null")
	}
	if config.LogStdErr {
		errPath := config.MakeReleasePath("releases/%s/logs/stderr.log")
		errOut, err := os.Create(errPath)
		if err != nil {
			config.Logger.Panic("Error setting up stderr: %v", err)
		}
		cmd.Stderr = errOut
		config.Logger.Info("stderr logged to %s", errPath)
	} else {
		config.Logger.Info("stderr logged to /dev/null")
	}
	notifier := make(chan os.Signal, 1)
	signal.Notify(notifier, syscall.SIGINT, syscall.SIGTERM)
	// Guarantee HOME is set in case we want to connect a remote
	// console to the running release
	cmd.Env = []string{fmt.Sprintf("HOME=%s", config.ReleaseRoot)}
	if config.Verbose {
		config.Logger.Info("Release command: '%s'", sanitizedCommand(cmd.String()))
	}
	go func() {
		err = cmd.Start()
		if err != nil {
			config.Logger.Panic("Error starting release: %v", err)
		} else {
			config.Logger.Info("Release running on OS process %d", cmd.Process.Pid)
		}
		cmd.Wait()
	}()
	for {
		s := <-notifier
		config.Logger.Info("Received '%v' signal. Shutting down %s...", s, config.ReleaseRoot)
		cmd.Process.Kill()
		os.Exit(0)
	}
}
