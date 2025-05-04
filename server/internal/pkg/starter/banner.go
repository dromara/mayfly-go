package starter

import (
	"fmt"
	"mayfly-go/internal/pkg/config"
	"mayfly-go/pkg/logx"
	"os"
	"runtime/debug"
)

func printBanner() {
	buildInfo, _ := debug.ReadBuildInfo()
	logx.Print(fmt.Sprintf(`
                        __ _                         
 _ __ ___   __ _ _   _ / _| |_   _        __ _  ___  
| '_ ' _ \ / _' | | | | |_| | | | |_____ / _' |/ _ \ 
| | | | | | (_| | |_| |  _| | |_| |_____| (_| | (_) |   version: %s | go_version: %s | pid: %d
|_| |_| |_|\__,_|\__, |_| |_|\__, |      \__, |\___/ 
                 |___/       |___/       |___/       `, config.Version, buildInfo.GoVersion, os.Getpid()))
}
