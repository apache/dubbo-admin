package component

import (
	"sync"

	"github.com/kumahq/kuma/pkg/core"
	"github.com/kumahq/kuma/pkg/util/channels"
)

var log = core.Log.WithName("bootstrap")

// Component defines a process that will be run in the application
// Component should be designed in such a way that it can be stopped by stop channel and started again (for example when instance is reelected for a leader).
type Component interface {
	// Start blocks until the channel is closed or an error occurs.
	// The component will stop running when the channel is closed.
	Start(<-chan struct{}) error

	// NeedLeaderElection indicates if component should be run only by one instance of Control Plane even with many Control Plane replicas.
	NeedLeaderElection() bool
}

// GracefulComponent is a component that supports waiting until it's finished.
// It's useful if there is cleanup logic that has to be executed before the process exits
// (i.e. sending SIGTERM signals to subprocesses started by this component).
type GracefulComponent interface {
	Component

	// WaitForDone blocks until all components are done.
	// If a component was not started (i.e. leader components on non-leader CP) it returns immediately.
	WaitForDone()
}

// Component of Kuma, i.e. gRPC Server, HTTP server, reconciliation loop.
var _ Component = ComponentFunc(nil)

type ComponentFunc func(<-chan struct{}) error

func (f ComponentFunc) NeedLeaderElection() bool {
	return false
}

func (f ComponentFunc) Start(stop <-chan struct{}) error {
	return f(stop)
}

var _ Component = LeaderComponentFunc(nil)

type LeaderComponentFunc func(<-chan struct{}) error

func (f LeaderComponentFunc) NeedLeaderElection() bool {
	return true
}

func (f LeaderComponentFunc) Start(stop <-chan struct{}) error {
	return f(stop)
}

type Manager interface {

	// Add registers a component, i.e. gRPC Server, HTTP server, reconciliation loop.
	Add(...Component) error

	// Start starts registered components and blocks until the Stop channel is closed.
	// Returns an error if there is an error starting any component.
	// If there are any GracefulComponent, it waits until all components are done.
	Start(<-chan struct{}) error
}

var _ Manager = &manager{}

func NewManager(leaderElector LeaderElector) Manager {
	return &manager{
		leaderElector: leaderElector,
	}
}

type manager struct {
	components    []Component
	leaderElector LeaderElector
}

func (cm *manager) Add(c ...Component) error {
	cm.components = append(cm.components, c...)
	return nil
}

func (cm *manager) waitForDone() {
	for _, c := range cm.components {
		if gc, ok := c.(GracefulComponent); ok {
			gc.WaitForDone()
		}
	}
}

func (cm *manager) Start(stop <-chan struct{}) error {
	errCh := make(chan error)

	cm.startNonLeaderComponents(stop, errCh)
	cm.startLeaderComponents(stop, errCh)

	defer cm.waitForDone()
	select {
	case <-stop:
		return nil
	case err := <-errCh:
		return err
	}
}

func (cm *manager) startNonLeaderComponents(stop <-chan struct{}, errCh chan error) {
	for _, component := range cm.components {
		if !component.NeedLeaderElection() {
			go func(c Component) {
				if err := c.Start(stop); err != nil {
					errCh <- err
				}
			}(component)
		}
	}
}

func (cm *manager) startLeaderComponents(stop <-chan struct{}, errCh chan error) {
	// leader stop channel needs to be stored in atomic because it will be written by leader elector goroutine
	// and read by the last goroutine in this function.
	// we need separate channel for leader components because they can be restarted
	mutex := sync.Mutex{}
	leaderStopCh := make(chan struct{})
	closeLeaderCh := func() {
		mutex.Lock()
		defer mutex.Unlock()
		if !channels.IsClosed(leaderStopCh) {
			close(leaderStopCh)
		}
	}

	cm.leaderElector.AddCallbacks(LeaderCallbacks{
		OnStartedLeading: func() {
			log.Info("leader acquired")
			mutex.Lock()
			defer mutex.Unlock()
			leaderStopCh = make(chan struct{})
			for _, component := range cm.components {
				if component.NeedLeaderElection() {
					go func(c Component) {
						if err := c.Start(leaderStopCh); err != nil {
							errCh <- err
						}
					}(component)
				}
			}
		},
		OnStoppedLeading: func() {
			log.Info("leader lost")
			closeLeaderCh()
		},
	})
	go cm.leaderElector.Start(stop)
	go func() {
		<-stop
		closeLeaderCh()
	}()
}
