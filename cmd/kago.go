package main

import (
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/kamalshkeir/kago/core/utils/katcher"
)

func main() {
	root := flag.String("root","","root is fullPath to the project")
	watch := flag.String("watch","","directory to watch inside root,if empty, will take all files and dirs inside root")
	every := flag.Int("every",313,"time in milliseconds")
	flag.Parse()
	if root == nil || *root == "" {
		fmt.Println("error: root tag not specified")
		return
	}
	sp := strings.Split(*watch,"/")
	katcher.Watch(time.Duration(*every) * time.Millisecond,*root,sp...)
}