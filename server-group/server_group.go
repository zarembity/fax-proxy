package server_group

import (
	"context"
	"fmt"
	"sync/atomic"

	log "github.com/sirupsen/logrus"
)

const (
	cSGStateIdle     = 0
	cSGStateStarting = 1
	cSGStateRunning  = 2
	cSGStateStopped  = 3
)

// Server is an object, which starts some processing in Serve.
// Processing can be stopped by calling Stop.
type Server interface {
	Serve() error
	Stop(ctx context.Context) error
}

// ServerGroup is a group of servers running simultaneously.
// They start working by calling StartAll and can be stopped by calling StopAll.
// If one of the servers stops working, StopAll is called automatically.
type ServerGroup struct {
	servers []serverInfo
	errChan chan serverRunResult
	resChan chan error
	state   int32
}

type serverInfo struct {
	s        Server
	stopped  int32
	critical bool
}

type serverRunResult struct {
	err error
	id  int
}

// NewServerGroup creates a new ServerGroup.
func NewServerGroup() *ServerGroup {
	return &ServerGroup{}
}

// Add add a critical server. It must be called before StartAll.
func (sg *ServerGroup) Add(s Server) {
	sg.AddWithCritical(s, true)
}

// AddWithCritical add a server and allows to specify 'critical' flag.
// If set, the group will be stopped after server's failure.
// If not, the group continues running.
// It must be called before StartAll.
func (sg *ServerGroup) AddWithCritical(s Server, critical bool) {
	sg.servers = append(sg.servers, serverInfo{s: s, critical: critical})
}

// StartAll starts all the servers.
func (sg *ServerGroup) StartAll() error {
	if !atomic.CompareAndSwapInt32(&sg.state, cSGStateIdle, cSGStateStarting) {
		return fmt.Errorf("invalid group state. already started?")
	}

	sg.errChan = make(chan serverRunResult, len(sg.servers))
	sg.resChan = make(chan error)
	for idx, si := range sg.servers {
		go func(s Server, idx int) {
			sg.errChan <- serverRunResult{err: s.Serve(), id: idx}
		}(si.s, idx)
	}
	go sg.waitLoop()
	atomic.StoreInt32(&sg.state, cSGStateRunning)
	return nil
}

// StopAll stops all the servers.
func (sg *ServerGroup) StopAll() error {
	if !atomic.CompareAndSwapInt32(&sg.state, cSGStateRunning, cSGStateStopped) {
		return fmt.Errorf("invalid group state. already stopped?")
	}
	for _, si := range sg.servers {
		if !atomic.CompareAndSwapInt32(&si.stopped, 0, 1) {
			continue
		}
		if err := si.s.Stop(context.Background()); err != nil {
			log.Errorf("server stop error: %v\n", err)
		}
	}
	return nil
}

// WaitAll wait for servers to finish their job.
func (sg *ServerGroup) WaitAll() error {
	err, ok := <-sg.resChan
	if !ok {
		return fmt.Errorf("WaitAll has already been called")
	}
	return err
}

func (sg *ServerGroup) waitLoop() {
	var result error
	for range sg.servers {
		runRes := <-sg.errChan
		atomic.StoreInt32(&sg.servers[runRes.id].stopped, 1)
		if runRes.err != nil {
			log.Errorf("server error: %v", runRes.err)
			result = runRes.err
		}
		// StopAll checks if the servers are still running, and if they are, stops them all.
		if sg.servers[runRes.id].critical {
			sg.StopAll()
		}
	}
	sg.resChan <- result
	close(sg.resChan)
}

// StartServerAsync calls s.Serve in a separate goroutine.
// The result error will be sent to ch.
func StartServerAsync(s Server, ch chan<- error) {
	go func() {
		err := s.Serve()
		if ch != nil {
			ch <- err
		} else if err != nil {
			log.Errorf("server error: %v", err)
		}
	}()
}
