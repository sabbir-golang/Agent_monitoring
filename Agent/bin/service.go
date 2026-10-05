package main

import (
	"log"
	"os"
	"time"

	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/debug"
)

const serviceName = "GoBackgroundService"

type myService struct{}

func (m *myService) Execute(args []string, r <-chan svc.ChangeRequest, status chan<- svc.Status) (bool, uint32) {

	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown | svc.AcceptPauseAndContinue

	status <- svc.Status{State: svc.StartPending}

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	status <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

loop:
	for {
		select {

		case <-ticker.C:
			// 🔁 MAIN BACKGROUND WORK GOES HERE
			// Example: log heartbeat
			log.Println("service running...")

		case c := <-r:
			switch c.Cmd {

			case svc.Stop, svc.Shutdown:
				break loop

			case svc.Pause:
				status <- svc.Status{State: svc.Paused, Accepts: cmdsAccepted}

			case svc.Continue:
				status <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

			default:
				log.Printf("unexpected control request #%d", c)
			}
		}
	}

	status <- svc.Status{State: svc.StopPending}
	return false, 0
}

func run(isDebug bool) {
	var err error

	if isDebug {
		err = debug.Run(serviceName, &myService{})
	} else {
		err = svc.Run(serviceName, &myService{})
	}

	if err != nil {
		log.Fatal(err)
	}
}

func setupLogging() {
	f, err := os.OpenFile("service.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	log.SetOutput(f)
}

func main() {
	setupLogging()

	isInteractive, err := svc.IsAnInteractiveSession()
	if err != nil {
		log.Fatal(err)
	}

	run(isInteractive)
}
