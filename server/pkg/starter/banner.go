package starter

import (
	"fmt"
	"mayfly-go/pkg/config"
	"mayfly-go/pkg/global"
)

func printBanner() {
	global.Log.Print(fmt.Sprintf(`
                        __ _                         
 _ __ ___   __ _ _   _ / _| |_   _        __ _  ___  
| '_ ' _ \ / _' | | | | |_| | | | |_____ / _' |/ _ \ 
| | | | | | (_| | |_| |  _| | |_| |_____| (_| | (_) |   version: %s
|_| |_| |_|\__,_|\__, |_| |_|\__, |      \__, |\___/ 
                 |___/       |___/       |___/       `, config.Version))
}
