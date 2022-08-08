package main

import (
	"flag"
	"strings"
	"time"

	"github.com/kamalshkeir/kago/core/utils/katcher"
	"github.com/kamalshkeir/kago/core/utils/logger"
)

func main() {
	root := flag.String("root","","root is fullPath to the project")
	watch := flag.String("watch","","directory to watch inside root,if empty, will take all files and dirs inside root")
	every := flag.Int("every",313,"time in milliseconds")
	flag.Parse()
	if root == nil || *root == "" {
		logger.Printfs("rderror: root tag not specified")
		return
	}
	sp := strings.Split(*watch,"/")
	katcher.Watch(time.Duration(*every) * time.Millisecond,*root,sp...)
}