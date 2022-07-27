package logger

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/kamalshkeir/kago/core/settings"
	"github.com/kamalshkeir/kago/core/utils/eventbus"
)

const (
	Red     = "\033[1;31m%v\033[0m\n"
	Green   = "\033[1;32m%v\033[0m\n"
	Yellow  = "\033[1;33m%v\033[0m\n"
	Blue    = "\033[1;34m%v\033[0m\n"
	Magenta = "\033[5;35m%v\033[0m\n"
)

var StreamLogs = []string{}

func init() {
	eventbus.Subscribe("internal-logs",func(_ map[string]string) {
		lenStream := len(StreamLogs)
		if lenStream > 30 {
			StreamLogs = StreamLogs[lenStream-20:]
		}
	})
}

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
		caller := runtime.FuncForPC(pc).Name()
		fmt.Printf("\033[1;31m [ERROR] %s [line:%d] : %v \033[0m \n", caller, line, err)
		if settings.GlobalConfig.Logs {
			StreamLogs = append(StreamLogs, fmt.Sprintf("[ERROR] %s [line:%d] : %v \n", caller, line, err))
			eventbus.Publish("internal-logs",map[string]string{})
		}
		return true
	}
	return false
}

// Error println anything with red color 
func Error(anything ...interface{}) {
	pc, _, line, _ := runtime.Caller(1)
	caller := runtime.FuncForPC(pc).Name()
	placeholder := strings.Repeat("%v,",len(anything))
	ph := strings.Replace(placeholder[:len(placeholder)-1],",","  ",-1) 
	new := fmt.Sprintf("\033[1;31m [ERROR] %s [line:%d] : %s \033[0m \n", caller, line, ph)
	if settings.GlobalConfig.Logs {
		StreamLogs = append(StreamLogs, fmt.Sprintf("[ERROR] %s [line:%d] : %v \n", caller, line, fmt.Sprintf(ph,anything...)))
		eventbus.Publish("internal-logs",map[string]string{})
	}
	fmt.Printf(new,anything...)
}

// Info println anything with blue color 
func Info(anything ...interface{}) {
	pc, _, line, _ := runtime.Caller(1)
	caller := runtime.FuncForPC(pc).Name()
	placeholder := strings.Repeat("%v,",len(anything))
	ph := strings.Replace(placeholder[:len(placeholder)-1],",","  ",-1) 
	new := fmt.Sprintf("\033[1;34m [INFO] %s [line:%d] : %s \033[0m \n", caller, line, ph)
	if settings.GlobalConfig.Logs {
		StreamLogs = append(StreamLogs, fmt.Sprintf("[INFO] %s [line:%d] : %v \n", caller, line, fmt.Sprintf(ph,anything...)))
		eventbus.Publish("internal-logs",map[string]string{})
	}
	fmt.Printf(new,anything...)
}

// Debug println anything with blue color 
func Debug(anything ...interface{}) {
	pc, _, line, _ := runtime.Caller(1)
	caller := runtime.FuncForPC(pc).Name()
	placeholder := strings.Repeat("%v,",len(anything))
	ph := strings.Replace(placeholder[:len(placeholder)-1],",","  ",-1) 
	new := fmt.Sprintf("\033[1;34m [DEBUG] %s [line:%d] : %s \033[0m \n", caller, line, ph)
	if settings.GlobalConfig.Logs {
		StreamLogs = append(StreamLogs, fmt.Sprintf("[DEBUG] %s [line:%d] : %v \n", caller, line, fmt.Sprintf(ph,anything...)))
		eventbus.Publish("internal-logs",map[string]string{})
	}
	fmt.Printf(new,anything...)
}

// Success println anything with green color 
func Success(anything ...interface{}) {
	pc, _, line, _ := runtime.Caller(1)
	caller := runtime.FuncForPC(pc).Name()
	placeholder := strings.Repeat("%v,",len(anything))
	ph := strings.Replace(placeholder[:len(placeholder)-1],",","  ",-1) 
	new := fmt.Sprintf("\033[1;32m [SUCCESS] %s [line:%d] : %s \033[0m \n", caller, line, ph)
	if settings.GlobalConfig.Logs {
		StreamLogs = append(StreamLogs, fmt.Sprintf("[SUCCESS] %s [line:%d] : %v \n", caller, line, fmt.Sprintf(ph,anything...)))
		eventbus.Publish("internal-logs",map[string]string{})
	}
	fmt.Printf(new,anything...)
}

// Warning println anything with yellow color 
func Warn(anything ...interface{}) {
	pc, _, line, _ := runtime.Caller(1)
	caller := runtime.FuncForPC(pc).Name()
	placeholder := strings.Repeat("%v,",len(anything))
	ph := strings.Replace(placeholder[:len(placeholder)-1],",","  ",-1)
	new := fmt.Sprintf("\033[1;35m [WARN] %s [line:%d] : %s \033[0m \n",caller, line, ph)
	if settings.GlobalConfig.Logs {
		StreamLogs = append(StreamLogs, fmt.Sprintf("[WARN] %s [line:%d] : %v \n", caller, line, fmt.Sprintf(ph,anything...)))
		eventbus.Publish("internal-logs",map[string]string{})
	}
	fmt.Printf(new,anything...)
}







var Ascii1 string = `                                                                                                                    

 ░░░░░░  ░░░░░░░░ ░░░░░░░░░░░░░░ ░░░░░░░░░░░░░░ ░░░░░░░░░░░░░░
 ░░▄▀░░  ░░▄▀▄▀░░ ░░▄▀▄▀▄▀▄▀▄▀░░ ░░▄▀▄▀▄▀▄▀▄▀░░ ░░▄▀▄▀▄▀▄▀▄▀░░
 ░░▄▀░░  ░░▄▀░░░░ ░░▄▀░░░░░░▄▀░░ ░░▄▀░░░░░░░░░░ ░░▄▀░░░░░░▄▀░░
 ░░▄▀░░  ░░▄▀░░   ░░▄▀░░  ░░▄▀░░ ░░▄▀░░         ░░▄▀░░  ░░▄▀░░
 ░░▄▀░░░░░░▄▀░░   ░░▄▀░░░░░░▄▀░░ ░░▄▀░░         ░░▄▀░░  ░░▄▀░░
 ░░▄▀▄▀▄▀▄▀▄▀░░   ░░▄▀▄▀▄▀▄▀▄▀░░ ░░▄▀░░  ░░░░░░ ░░▄▀░░  ░░▄▀░░
 ░░▄▀░░░░░░▄▀░░   ░░▄▀░░░░░░▄▀░░ ░░▄▀░░  ░░▄▀░░ ░░▄▀░░  ░░▄▀░░
 ░░▄▀░░  ░░▄▀░░   ░░▄▀░░  ░░▄▀░░ ░░▄▀░░  ░░▄▀░░ ░░▄▀░░  ░░▄▀░░
 ░░▄▀░░  ░░▄▀░░░░ ░░▄▀░░  ░░▄▀░░ ░░▄▀░░░░░░▄▀░░ ░░▄▀░░░░░░▄▀░░
 ░░▄▀░░  ░░▄▀▄▀░░ ░░▄▀░░  ░░▄▀░░ ░░▄▀▄▀▄▀▄▀▄▀░░ ░░▄▀▄▀▄▀▄▀▄▀░░
 ░░░░░░  ░░░░░░░░ ░░░░░░  ░░░░░░ ░░░░░░░░░░░░░░ ░░░░░░░░░░░░░░ V1.0.0
`
var Ascii2 string = `
 _    __  ______   ______   ______  
| |  / / | |  | | | | ____ / |  | \ 
| |-< <  | |__| | | |  | | | |  | | 
|_|  \_\ |_|  |_| |_|__|_| \_|__|_/ V1.0.0                                             
`
var Ascii3 string = `                                                                          
                               ___              
  .'|   .'|   .'|=|'.     .'|=|_.'    .'|=|'.   
.'  | .' .' .'  | |  '. .'  |___    .'  | |  '. 
|   |=|.:   |   |=|   | |   |'._|=. |   | |   | 
|   |   |'. |   | |   | '.  |  __|| '.  | |  .' 
|___|   |_| |___| |___|   '.|=|_.''   '.|=|.'   V1.0.0                                                                          	   
`
var Ascii4 string = `                                                                                                                    
 ___   _  _______  _______  _______ 
|   | | ||   _   ||       ||       |
|   |_| ||  |_|  ||    ___||   _   |
|      _||       ||   | __ |  | |  |
|     |_ |       ||   ||  ||  |_|  |
|    _  ||   _   ||   |_| ||       |
|___| |_||__| |__||_______||_______| V1.0.0
`
var Ascii5 string = `
##  ###   ######    ####    #####   
## ####  #######   ######  #######  
####     ###  ##  ### ##   ### ###  
######   ##   ##  ##       ##   ##  
#######  ## ####  ##  ###  ##   ##  
##  ###  ##   ##   ##  ##  ### ###  
##   ##   ##  ##    ####    #####   V1.0.0
`
var Ascii6 string = `                                                                          
8 8888     ,88'          .8.              ,o888888o.        ,o888888o.     
8 8888    ,88'          .888.            8888     '88.   . 8888     '88.   
8 8888   ,88'          :88888.        ,8 8888       '8. ,8 8888       '8b  
8 8888  ,88'          . '88888.       88 8888           88 8888        '8b 
8 8888 ,88'          .8. '88888.      88 8888           88 8888         88 
8 8888 88'          .8'8. '88888.     88 8888           88 8888         88 
8 888888<          .8' '8. '88888.    88 8888   8888888 88 8888        ,8P 
8 8888 'Y8.       .8'   '8. '88888.   '8 8888       .8' '8 8888       ,8P  
8 8888   'Y8.    .888888888. '88888.     8888     ,88'   ' 8888     ,88'   
8 8888     'Y8. .8'       '8. '88888.     '8888888P'        '8888888P'                     
`
var Ascii7 string = `                                                                                               
 _     _           _______         
(_)   | |         (_______)        
 _____| |  _____   _   ___    ___  
|  _   _) (____ | | | (_  |  / _ \ 
| |  \ \  / ___ | | |___) | | |_| |
|_|   \_) \_____|  \_____/   \___/ 
`
var Ascii8 string = `                               
  o                                            
 <|>      o/                                     
 / \    o/                                      
 \o/  o/     o__ __o/    o__ __o/    o__ __o   
  |  /      /v     |    /v     |    /v     v\  
 / \/>     />     / \  />     / \  />       <\ 
 \o/\o     \      \o/  \      \o/  \         / 
  |  v\     o      |    o      |    o       o  
 / \  <\    <\__  / \   <\__  < >   <\__ __/>  
                               |               
                       o__     o               
                       <\__ __/>      V1.0.0
`
var Ascii9 string = `                               
888  /              e88~~\           
888 /      /~~~8e  d888      e88~-_  
888/\          88b 8888 __  d888   i 
888  \    e88~-888 8888   | 8888   | 
888   \  C888  888 Y888   | Y888   ' 
888    \  "88_-888  "88__/   "88_-~  V1.0.0                                            
`
var Ascii10 string = `                                                                                               
 ____  __.           ________           
|    |/ _| _____    /  _____/    ____   
|      <   \__  \  /   \  ___   /  _ \  
|    |  \   / __ \_\    \_\  \ (  <_> ) 
|____|__ \ (____  / \______  /  \____/  
        \/      \/         \/           V1.0.0
`


func GetLastestTag() string {
	cmd := exec.Command("git", "describe", "--tags","--abbrev=0")
	out,err := cmd.Output()
	if CheckError(err) {
		return ""
	}
	return string(out)
}
