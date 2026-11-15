package main

import (
	"golang.org/x/sys/windows/svc"
	"log"
	"time"
)

type myService struct{}

func (m *myService) Execute(args []string, r <-chan svc.ChangeRequest, status chan<- svc.Status) (bool, uint32) {

	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown // | svc.AcceptPauseAndContinue
	tick := time.Tick(5 * time.Second)

	status <- svc.Status{State: svc.StartPending}
	go runExporterNormal()
	status <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

loop:
	for {
		select {
		case <-tick:
			// log.Print("Tick Handled...!")
			collectMetrics()
			// collectPrinterMetrics()
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				status <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				log.Print("Shutting service...!")
				break loop
			case svc.Pause:
				status <- svc.Status{State: svc.Paused, Accepts: cmdsAccepted}
			case svc.Continue:
				status <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
			default:
				log.Printf("Unexpected service control request #%d", c)
			}
		}
	}

	status <- svc.Status{State: svc.StopPending}
	return false, 1
}

func main() {
	isService, err := svc.IsWindowsService()
	if err != nil {
		log.Fatalf("failed to determine if we are in an interactive session: %v", err)
	}

	if isService {
		// запускаемся как сервис
		err = svc.Run("_olymp_exporter", &myService{})
		if err != nil {
			log.Fatalf("service failed: %v", err)
		}
		return
	}

	// если запущено интерактивно, запускаем как обычное приложение
	go runExporterNormal()
	for {
		collectMetrics()
		// collectPrinterMetrics()
		time.Sleep(5 * time.Second)
	}
}
