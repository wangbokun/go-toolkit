package process

import (
	"context"
	"time"

	"github.com/mylxsw/go-toolkit/log"
)

// Manager is process manager
type Manager struct {
	programs          map[string]*Program
	restartProcess    chan *Process
	closeTimeout      time.Duration
	processOutputFunc OutputFunc
}

// NewManager create a new process manager
func NewManager(closeTimeout time.Duration, processOutputFunc OutputFunc) *Manager {
	return &Manager{
		programs:          make(map[string]*Program),
		restartProcess:    make(chan *Process),
		closeTimeout:      closeTimeout,
		processOutputFunc: processOutputFunc,
	}
}

func (manager *Manager) AddProgram(name string, command string, procNum int, username string) {
	manager.programs[name] = NewProgram(name, command, username, procNum).initProcesses(manager.processOutputFunc)
}

func (manager *Manager) Watch(ctx context.Context) {
	for _, program := range manager.programs {
		for _, proc := range program.processes {
			go manager.startProcess(proc, 0)
		}
	}

	for {
		select {
		case process := <-manager.restartProcess:
			go manager.startProcess(process, process.retryDelayTime())
		case <-ctx.Done():
			for _, program := range manager.programs {
				for _, proc := range program.processes {
					proc.stop(manager.closeTimeout)
				}
			}
			return
		}
	}
}

func (manager *Manager) startProcess(process *Process, delay time.Duration) {
	if delay > 0 {
		log.Module("process").Debugf("process %s will start after %.2fs", process.Name, delay.Seconds())
	}

	process.lock.Lock()
	defer process.lock.Unlock()

	process.timer = time.AfterFunc(delay, func() {
		process.removeTimer()

		log.Module("process").Debugf("process %s starting...", process.Name)
		manager.restartProcess <- <-process.start()
	})

}

// Programs return all programs
func (manager *Manager) Programs() map[string]*Program {
	return manager.programs
}