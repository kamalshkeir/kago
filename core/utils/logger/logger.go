package logger

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

const (
	Red     = "\033[1;31m%s\033[0m\n"
	Green   = "\033[1;32m%s\033[0m\n"
	Yellow  = "\033[1;33m%s\033[0m\n"
	Blue    = "\033[1;34m%s\033[0m\n"
	Magenta = "\033[5;35m%s\033[0m\n"
)

// Printf take pattern(rd,gr,yl,bl,mg), varsString, varsValues
func Printf(pattern string,anything ...interface{}) {
	pc, _, line, _ := runtime.Caller(1)
	
	var colorCode int
	var colorUsed = true
	switch pattern[:2] {
		case "rd":
			colorCode=31
		case "gr":
			colorCode=32
		case "yl":
			colorCode=33
		case "bl":
			colorCode=34
		case "mg":
			colorCode=35
		default:
			colorUsed = false
			colorCode=34
	}
	if colorUsed {
		pattern =  fmt.Sprintf("\033[1;%dm %s[line:%d]: %s \033[0m \n",colorCode,runtime.FuncForPC(pc).Name(),line,pattern[2:])
	} else {
		pattern =  fmt.Sprintf("\033[1;%dm %s[line:%d]: %s \033[0m \n",colorCode,runtime.FuncForPC(pc).Name(),line,pattern)
	}
	fmt.Printf(pattern,anything...)
}

// CheckError check if err not nil print it and return true
func CheckError(err error) bool {
	if err != nil {
		pc, _, line, _ := runtime.Caller(1)
		fmt.Printf("\033[1;31m [error] %s [line:%d] : %v \033[0m \n", runtime.FuncForPC(pc).Name(), line, err)
		return true
	}
	return false
}

// Error println anything with red color 
func Error(anything ...interface{}) {
	pc, _, line, _ := runtime.Caller(1)
	placeholder := strings.Repeat("%v,",len(anything))
	ph := strings.Replace(placeholder[:len(placeholder)-1],",","  ",-1) 
	new := fmt.Sprintf("\033[1;31m [ERROR] %s [line:%d] : %s \033[0m \n", runtime.FuncForPC(pc).Name(), line, ph)
	fmt.Printf(new,anything...)
}

// Info println anything with blue color 
func Info(anything ...interface{}) {
	pc, _, line, _ := runtime.Caller(1)
	placeholder := strings.Repeat("%v,",len(anything))
	ph := strings.Replace(placeholder[:len(placeholder)-1],",","  ",-1) 
	new := fmt.Sprintf("\033[1;34m [INFO] %s [line:%d] : %s \033[0m \n", runtime.FuncForPC(pc).Name(), line, ph)
	fmt.Printf(new,anything...)
}

// Info println anything with blue color 
func Debug(anything ...interface{}) {
	pc, _, line, _ := runtime.Caller(1)
	placeholder := strings.Repeat("%v,",len(anything))
	ph := strings.Replace(placeholder[:len(placeholder)-1],",","  ",-1) 
	new := fmt.Sprintf("\033[1;34m [Debug] %s [line:%d] : %s \033[0m \n", runtime.FuncForPC(pc).Name(), line, ph)
	fmt.Printf(new,anything...)
}

// Success println anything with green color 
func Success(anything ...interface{}) {
	pc, _, line, _ := runtime.Caller(1)
	placeholder := strings.Repeat("%v,",len(anything))
	ph := strings.Replace(placeholder[:len(placeholder)-1],",","  ",-1) 
	new := fmt.Sprintf("\033[1;32m [SUCCESS] %s [line:%d] : %s \033[0m \n", runtime.FuncForPC(pc).Name(), line, ph)
	fmt.Printf(new,anything...)
}

// Warning println anything with yellow color 
func Warn(anything ...interface{}) {
	pc, _, line, _ := runtime.Caller(1)
	placeholder := strings.Repeat("%v,",len(anything))
	ph := strings.Replace(placeholder[:len(placeholder)-1],",","  ",-1)
	new := fmt.Sprintf("\033[1;35m [WARNING] %s [line:%d] : %s \033[0m \n", runtime.FuncForPC(pc).Name(), line, ph)
	fmt.Printf(new,anything...)
}




var Ascii string = `                                                                                                                    
                               ___              
  .'|   .'|   .'|=|'.     .'|=|_.'    .'|=|'.   
.'  | .' .' .'  | |  '. .'  |___    .'  | |  '. 
|   |=|.:   |   |=|   | |   |'._|=. |   | |   | 
|   |   |'. |   | |   | '.  |  __|| '.  | |  .' 
|___|   |_| |___| |___|   '.|=|_.''   '.|=|.'   v1.0.0   
`

var Ascii2 string = `
 d8b                                   
 ?88                                   
  88b                                  
  888  d88' d888b8b   d888b8b   d8888b 
  888bd8P' d8P' ?88  d8P' ?88  d8P' ?88
 d88888b   88b  ,88b 88b  ,88b 88b  d88
d88' '?88b,'?88P''88b'?88P''88b'?8888P'
                            )88        
                           ,88P        
                       '?8888P                                                         
`

var Ascii3 string = `                                                                          
 __  __            ____              
/\ \/\ \          /\  _'\            
\ \ \/'/'     __  \ \ \L\_\    ___   
 \ \ , <    /'__'\ \ \ \L_L   / __'\ 
  \ \ \\'\ /\ \L\.\_\ \ \/, \/\ \L\ \
   \ \_\ \_\ \__/.\_\\ \____/\ \____/
    \/_/\/_/\/__/\/_/ \/___/  \/___/ 
                                                                          	   
`



func GetLastestTag() string {
	cmd := exec.Command("git", "describe", "--tags","--abbrev=0")
	out,err := cmd.Output()
	if CheckError(err) {
		return ""
	}
	return string(out)
}
