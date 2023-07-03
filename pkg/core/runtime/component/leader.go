package component

import "sync/atomic"

// LeaderCallbacks defines callbacks for events from LeaderElector
// It is guaranteed that each methods will be executed from the same goroutine, so only one method can be run at once.
type LeaderCallbacks struct {
	OnStartedLeading func()
	OnStoppedLeading func()
}

type LeaderElector interface {
	AddCallbacks(LeaderCallbacks)
	// IsLeader should be used for diagnostic reasons (metrics/API info), because there may not be any leader elector for a short period of time.
	// Use Callbacks to write logic to execute when Leader is elected.
	IsLeader() bool

	// Start blocks until the channel is closed or an error occurs.
	Start(stop <-chan struct{})
}

type LeaderInfo interface {
	IsLeader() bool
}

var _ LeaderInfo = &LeaderInfoComponent{}
var _ Component = &LeaderInfoComponent{}

type LeaderInfoComponent struct {
	leader int32
}

func (l *LeaderInfoComponent) Start(stop <-chan struct{}) error {
	l.setLeader(true)
	<-stop
	l.setLeader(false)
	return nil
}

func (l *LeaderInfoComponent) NeedLeaderElection() bool {
	return true
}

func (p *LeaderInfoComponent) setLeader(leader bool) {
	var value int32 = 0
	if leader {
		value = 1
	}
	atomic.StoreInt32(&p.leader, value)
}
func (p *LeaderInfoComponent) IsLeader() bool {
	return atomic.LoadInt32(&(p.leader)) == 1
}
