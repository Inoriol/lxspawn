package logger

import (
	"fmt"
	"os"
	"sync"

	"github.com/Inoriol/lxspawn/internal/output"
)

const (
	reset  = "\033[0m"
	red    = "\033[31m"
	green  = "\033[32m"
	yellow = "\033[33m"
	cyan   = "\033[36m"
	blue   = "\033[34m"
)

var (
	mu      sync.Mutex
	verbose = false
	quiet   = false
	noColor = false
)

func init() {
	_, noColor = os.LookupEnv("NO_COLOR")
}

func SetVerbose(v bool) {
	mu.Lock()
	defer mu.Unlock()
	verbose = v
}

func IsVerbose() bool {
	mu.Lock()
	defer mu.Unlock()
	return verbose
}

func SetQuiet(q bool) {
	mu.Lock()
	defer mu.Unlock()
	quiet = q
}

func IsQuiet() bool {
	mu.Lock()
	defer mu.Unlock()
	return quiet
}

func Print(msg string) {
	mu.Lock()
	q := quiet
	mu.Unlock()
	if q || output.IsJSONMode() {
		return
	}
	fmt.Println(msg)
}

func Printf(format string, args ...interface{}) {
	mu.Lock()
	q := quiet
	mu.Unlock()
	if q || output.IsJSONMode() {
		return
	}
	fmt.Printf(format, args...)
}

func colorize(colorCode, msg string) string {
	mu.Lock()
	n := noColor
	mu.Unlock()
	if n {
		return msg
	}
	if _, dynamic := os.LookupEnv("NO_COLOR"); dynamic {
		return msg
	}
	return fmt.Sprintf("%s%s%s", colorCode, msg, reset)
}

func Info(msg string) {
	if !IsVerbose() || IsQuiet() || output.IsJSONMode() {
		return
	}
	fmt.Println(colorize(cyan, "lxspawn: "+msg))
}

func Debug(msg string) {
	if !IsVerbose() || IsQuiet() || output.IsJSONMode() {
		return
	}
	fmt.Println(colorize(cyan, "lxspawn: [debug] "+msg))
}

func Success(msg string) {
	if IsQuiet() || output.IsJSONMode() {
		return
	}
	fmt.Println(colorize(green, "lxspawn: "+msg))
}

func Warn(msg string) {
	if IsQuiet() || output.IsJSONMode() {
		return
	}
	fmt.Println(colorize(yellow, "lxspawn: "+msg))
}

func Error(msg string) {
	if output.IsJSONMode() {
		return
	}
	fmt.Fprintln(os.Stderr, colorize(red, "lxspawn: "+msg))
}

func Action(msg string) {
	if IsQuiet() || output.IsJSONMode() {
		return
	}
	fmt.Println(colorize(blue, "lxspawn: "+msg))
}
