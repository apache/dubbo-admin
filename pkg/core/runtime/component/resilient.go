package component

import (
	"time"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
)

const (
	backoffTime = 5 * time.Second
)

type resilientComponent struct {
	log       logr.Logger
	component Component
}

func NewResilientComponent(log logr.Logger, component Component) Component {
	return &resilientComponent{
		log:       log,
		component: component,
	}
}

func (r *resilientComponent) Start(stop <-chan struct{}) error {
	r.log.Info("starting resilient component ...")
	for generationID := uint64(1); ; generationID++ {
		errCh := make(chan error, 1)
		go func(errCh chan<- error) {
			defer close(errCh)
			// recover from a panic
			defer func() {
				if e := recover(); e != nil {
					if err, ok := e.(error); ok {
						errCh <- err
					} else {
						errCh <- errors.Errorf("%v", e)
					}
				}
			}()

			errCh <- r.component.Start(stop)
		}(errCh)
		select {
		case <-stop:
			r.log.Info("done")
			return nil
		case err := <-errCh:
			if err != nil {
				r.log.WithValues("generationID", generationID).Error(err, "component terminated with an error")
			}
		}
		<-time.After(backoffTime)
	}
}

func (r *resilientComponent) NeedLeaderElection() bool {
	return r.component.NeedLeaderElection()
}
