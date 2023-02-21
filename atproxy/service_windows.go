package main

import (
	"os"
	"time"

	"github.com/reusee/e5"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/eventlog"
	"golang.org/x/sys/windows/svc/mgr"
)

const serviceName = "atproxy"

func init() {

	commands["install"] = func() {
		m, err := mgr.Connect()
		ce(err)
		defer m.Disconnect()
		s, err := m.OpenService(serviceName)
		if err == nil {
			s.Close()
			return
		}
		exePath, err := os.Executable()
		ce(err)
		s, err = m.CreateService(serviceName, exePath, mgr.Config{
			DisplayName: "ATPROXY",
		})
		ce(err)
		defer s.Close()
		ce(eventlog.InstallAsEventCreate(serviceName, eventlog.Error|eventlog.Warning|eventlog.Info),
			e5.Do(func() {
				s.Delete()
			}))
	}

	commands["uninstall"] = func() {
		m, err := mgr.Connect()
		ce(err)
		defer m.Disconnect()
		s, err := m.OpenService(serviceName)
		if err != nil {
			pt("not installed\n")
			return
		}
		defer s.Close()
		//_, err = s.Control(svc.Stop)
		//ce(err)
		ce(s.Delete())
		ce(eventlog.Remove(serviceName))
	}

	inService, err := svc.IsWindowsService()
	ce(err)
	if !inService {
		return
	}
	go func() {
		ce(svc.Run(serviceName, new(Service)))
	}()
}

type Service struct{}

var _ svc.Handler = new(Service)

func (s *Service) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown | svc.AcceptPauseAndContinue
	changes <- svc.Status{State: svc.StartPending}
	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
loop:
	for {
		select {
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
				// Testing deadlock from https://code.google.com/p/winsvc/issues/detail?id=4
				time.Sleep(100 * time.Millisecond)
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				// golang.org/x/sys/windows/svc.TestExample is verifying this output.
				globalWaitTree.Cancel()
				break loop
			case svc.Pause:
				changes <- svc.Status{State: svc.Paused, Accepts: cmdsAccepted}
			case svc.Continue:
				changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
			default:
			}
		}
	}
	changes <- svc.Status{State: svc.StopPending}
	return
}
