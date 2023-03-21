package configuration

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"path"
	"strings"
	"time"
)

const DEFAULT_CONF_FILE = "./daginit.conf"
const EV_CONF_FILE = "DAGINIT_CONF"
const EV_COOKIE = "DAGINIT_COOKIE"
const EV_LOG_STDERR = "DAGINIT_LOG_STDERR"
const EV_LOG_STDOUT = "DAGINIT_LOG_STDOUT"
const EV_RELEASE_ROOT = "DAGINIT_RELROOT"
const EV_RELEASE_VERSION = "DAGINIT_RELVSN"
const LOG_FLAGS = log.Ldate | log.Ltime

var cookieChars = []string{
	"a", "b", "c", "d", "e", "f",
	"g", "h", "i", "j", "k", "l",
	"m", "n", "o", "p", "q", "r",
	"s", "t", "u", "v", "w", "x",
	"y", "z",
	"A", "B", "C", "D", "E", "F",
	"G", "H", "I", "J", "K", "L",
	"M", "N", "O", "P", "Q", "R",
	"S", "T", "U", "V", "W", "X",
	"Y", "Z",
	"0", "1", "2", "3", "4", "5",
	"6", "7", "8", "9",
	"=", "!", "?"}

type Logger struct {
	outputLogger *log.Logger
	errorLogger  *log.Logger
}

type Configuration struct {
	Logger         *Logger
	Cookie         string `json:"cookie"`
	ReleaseRoot    string `json:"release_root"`
	ReleaseVersion string `json:"release_version"`
	LogStdOut      bool   `json:"log_stdout"`
	LogStdErr      bool   `json:"log_stderr"`
	Verbose        bool   `json:"verbose"`
}

func (l *Logger) Debug(format string, v ...any) {
	l.happyLog("DEBUG", format, v...)
}

func (l *Logger) Info(format string, v ...any) {
	l.happyLog("INFO", format, v...)
}

func (l *Logger) Warn(format string, v ...any) {
	l.sadLog("WARN", format, v...)
}

func (l *Logger) Error(format string, v ...any) {
	l.sadLog("ERR", format, v...)
}

func (l *Logger) Panic(format string, v ...any) {
	l.errorLogger.Panicf(format, v...)
}

func (l *Logger) happyLog(level, format string, v ...any) {
	formatted := fmt.Sprintf(format, v...)
	msg := fmt.Sprintf("%s: %s", level, formatted)
	l.outputLogger.Println(msg)
}

func (l *Logger) sadLog(level, format string, v ...any) {
	formatted := fmt.Sprintf(format, v...)
	msg := fmt.Sprintf("%s: %s", level, formatted)
	l.errorLogger.Println(msg)
}

func readFile(filePath string, logger *Logger) ([]byte, error) {
	fd, err := os.Open(filePath)
	if err != nil {
		logger.Info("Configuration file %s not found. Using default config values.", filePath)
		return []byte{}, err
	}
	defer fd.Close()
	contents, err := io.ReadAll(fd)
	if err != nil {
		logger.Error("Error reading configuration file %s: %v", filePath, err)
		return []byte{}, err
	}
	return contents, nil
}

func setupLogger() *Logger {
	result := Logger{
		outputLogger: log.New(os.Stdout, "", LOG_FLAGS),
		errorLogger:  log.New(os.Stderr, "", LOG_FLAGS),
	}
	return &result
}

func generateCookie(rg *rand.Rand, size int) string {
	cookie := ""
	for len(cookie) < size {
		cookie = fmt.Sprintf("%s%s", cookie, cookieChars[rg.Intn(len(cookieChars))])
	}
	return cookie
}

func defaultConfiguration(logger *Logger) *Configuration {
	rg := rand.New(rand.NewSource(time.Now().UnixMilli()))
	return &Configuration{
		Cookie:         generateCookie(rg, rg.Intn(21)+9),
		ReleaseRoot:    "./relroot",
		ReleaseVersion: "",
		LogStdOut:      false,
		LogStdErr:      false,
		Verbose:        false,
		Logger:         logger,
	}
}

func convertBoolean(value string) (bool, error) {
	value = strings.ToLower(value)
	var err error = nil
	var converted bool
	switch value {
	case "true":
		converted = true
	case "yes":
		converted = true
	case "t":
		converted = true
	case "y":
		converted = true
	case "1":
		converted = true
	case "false":
		converted = false
	case "no":
		converted = false
	case "f":
		converted = false
	case "n":
		converted = false
	case "0":
		converted = false
	default:
		err = fmt.Errorf("invalid boolean value: %s", value)
	}
	return converted, err
}

func applyEnvVars(configuration *Configuration) error {
	var err error = nil
	var converted bool
	names := []string{EV_COOKIE, EV_RELEASE_ROOT, EV_RELEASE_VERSION, EV_LOG_STDOUT, EV_LOG_STDERR}
	for _, name := range names {
		value, exists := os.LookupEnv(name)
		if exists {
			switch name {
			case EV_COOKIE:
				configuration.Cookie = value
			case EV_RELEASE_ROOT:
				configuration.ReleaseRoot = value
			case EV_RELEASE_VERSION:
				configuration.ReleaseVersion = value
			case EV_LOG_STDOUT:
				converted, err = convertBoolean(value)
				if err == nil {
					configuration.LogStdOut = converted
				}
			case EV_LOG_STDERR:
				converted, err = convertBoolean(value)
				if err == nil {
					configuration.LogStdErr = converted
				}
			}
		}
	}
	return err
}

func Load(configFile string) (*Configuration, error) {
	logger := setupLogger()
	if configFile == "" {
		var exists bool
		configFile, exists = os.LookupEnv(EV_CONF_FILE)
		if !exists {
			configFile = DEFAULT_CONF_FILE
		}
	}
	logger.Info("Using configuration file %s", configFile)
	contents, err := readFile(configFile, logger)
	if err != nil {
		return nil, err
	}
	configuration := defaultConfiguration(logger)
	if len(contents) > 0 {
		if err := json.Unmarshal(contents, configuration); err != nil {
			return nil, err
		}
	}
	if err := applyEnvVars(configuration); err != nil {
		return nil, err
	}
	return configuration, nil
}

func (c *Configuration) MakeReleasePath(fragment string) string {
	if strings.Contains(fragment, "%s") {
		fragment = fmt.Sprintf(fragment, c.ReleaseVersion)
	}
	return path.Join(c.ReleaseRoot, fragment)
}
