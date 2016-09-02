package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"runtime"

	"github.com/takama/daemon"
)

var port *string = flag.String("port", "127.0.0.1:4444", "web port")
var logs *string = flag.String("logs", "/var/log/croncli/", "log path")
var conf *string = flag.String("conf", "crontab.conf", "crontab config")
var stopCh chan bool = make(chan bool)
var startCh chan bool = make(chan bool)

type Service struct {
	daemon.Daemon
}

func (service *Service) Manage() (string, error) {

	usage := "Usage: myservice install | remove | start | stop | status"

	// if received any kind of command, do it
	if len(os.Args) > 1 {
		command := os.Args[1]
		switch command {
		case "install":
			return service.Install()
		case "remove":
			return service.Remove()
		case "start":
			return service.Start()
		case "stop":
			return service.Stop()
		case "status":
			return service.Status()
		default:
			return usage, nil
		}
	}

	go jobHandle()

	http.HandleFunc("/set", set)
	http.HandleFunc("/get", get)
	http.HandleFunc("/del", del)
	http.HandleFunc("/log", loger)
	http.HandleFunc("/load", load)
	http.HandleFunc("/stop", stop)
	http.HandleFunc("/start", start)
	http.HandleFunc("/status", status)

	startErr := http.ListenAndServe(*port, nil)
	if startErr != nil {
		fmt.Println("Start server failed.", startErr)
		os.Exit(1)
	}

	return usage, nil
}

const (
	RUN_LOG_POSTFIX = `run.log`
	SVR_LOG         = `sys.log`
	DATEFORMAT      = `20060102`
	TIMEFORMAT      = `2006-01-02 15:04:05`
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	flag.Parse()

	initLog()

	var dependencies = []string{"dummy.service"}

	loaded, loadErr := loadConf()
	if !loaded {
		fmt.Printf("Err %s exit.\n", loadErr)
		os.Exit(1)
	}

	srv, err := daemon.New("croncli", "croncli service", dependencies...)

	service := &Service{srv}
	status, err := service.Manage()

	if err != nil {
		sysLog.Println(status, "\nError: ", err)
		os.Exit(1)
	}
	fmt.Println(status)

}
