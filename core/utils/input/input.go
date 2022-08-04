package input

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"syscall"

	"golang.org/x/term"
)

const (
	Red     = "\033[1;31m%v\033[0m"
	Green   = "\033[1;32m%v\033[0m"
	Yellow  = "\033[1;33m%v\033[0m"
	Blue    = "\033[1;34m%v\033[0m"
	Magenta = "\033[5;35m%v\033[0m"
)

// Input
func Int(color, desc string) (int, error) {
	// print message
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf(color, desc)

	// read line
	out, err := reader.ReadString('\n')
	if err != nil {
		if errors.Is(err, io.EOF) {
			os.Exit(0)
		} else {
			return 0, err
		}
	}
	out = strings.Replace(out, "\r\n", "", -1)
	out = strings.TrimSpace(out)
	outInt, err := strconv.Atoi(out)
	if err != nil {
		return 0, err
	}
	return outInt, nil
}

// Input
func Bool(color, desc string) (bool, error) {
	// print message
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf(color, desc)
	// read line
	out, err := reader.ReadString('\n')
	if err != nil {
		if errors.Is(err, io.EOF) {
			os.Exit(0)
		} else {
			return false, err
		}
	}
	out = strings.Replace(out, "\r\n", "", -1)
	out = strings.TrimSpace(out)
	outBool, err := strconv.ParseBool(out)
	if err != nil {
		return false, err
	}
	return outBool, nil
}

// Input
func String(color, desc string) (string, error) {
	// print message
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf(color, desc)
	// read line
	out, err := reader.ReadString('\n')
	if err != nil {
		if errors.Is(err, io.EOF) {
			os.Exit(0)
		} else {
			return "", err
		}
	}
	out = strings.Replace(out, "\r\n", "", -1)
	out = strings.TrimSpace(out)
	return out, nil
}

// Input
func Input(color, desc string) string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf(color, desc)
	out, err := reader.ReadString('\n')
	if err != nil {
		if errors.Is(err, io.EOF) {
			os.Exit(0)
		} else {
			return ""
		}
	}
	out = strings.Replace(out, "\r\n", "", -1)
	return strings.TrimSpace(out)
}

func Hidden(color, desc string) string {
	fmt.Printf(color, desc)
	out, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		if errors.Is(err, io.EOF) {
			os.Exit(0)
		} else {
			return ""
		}
	}
	res := string(out)
	res = strings.Replace(res, "\r\n", "", -1)
	fmt.Println(" ")
	return strings.TrimSpace(res)
}

var clear map[string]func() //create a map for storing clear funcs
func Clear() {
	prepareClear()
	value, ok := clear[runtime.GOOS] //runtime.GOOS -> linux, windows, darwin etc.
	if ok {                          //if we defined a clear func for that platform:
		value() //we execute it
	} else { //unsupported platform
		panic("Your platform is unsupported! I can't clear terminal screen :(")
	}
}

func prepareClear() {
	clear = make(map[string]func()) //Initialize it
	clear["linux"] = func() {
		cmd := exec.Command("clear") //Linux example, its tested
		cmd.Stdout = os.Stdout
		cmd.Run()
	}
	clear["windows"] = func() {
		cmd := exec.Command("cmd", "/c", "cls") //Windows example, its tested
		cmd.Stdout = os.Stdout
		cmd.Run()
	}
}
