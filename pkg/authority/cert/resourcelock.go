package cert

import (
	"context"
	"time"

	"github.com/apache/dubbo-admin/pkg/authority/config"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
)

func Resourcelock(storage Storage, options *config.Options, kubeClient *kubernetes.Clientset) error {
	identity := options.ResourcelockIdentity
	rlConfig := resourcelock.ResourceLockConfig{
		Identity: identity,
	}
	namespace := options.Namespace
	_, err := kubeClient.CoreV1().Namespaces().Get(context.TODO(), namespace, metav1.GetOptions{})
	if err != nil {
		namespace = "default"
	}
	lock, err := resourcelock.New(resourcelock.ConfigMapsLeasesResourceLock, namespace, "dubbo-lock-cert", kubeClient.CoreV1(), kubeClient.CoordinationV1(), rlConfig)
	if err != nil {
		return err
	}
	leaderElectionConfig := leaderelection.LeaderElectionConfig{
		Lock:          lock,
		LeaseDuration: 15 * time.Second,
		RenewDeadline: 10 * time.Second,
		RetryPeriod:   2 * time.Second,
		Callbacks: leaderelection.LeaderCallbacks{
			// leader
			OnStartedLeading: func(ctx context.Context) {
				// lock if multi server，refresh signed cert
				storage.SetAuthorityCert(GenerateAuthorityCert(storage.GetRootCert(), options.CaValidity))
			},
			// not leader
			OnStoppedLeading: func() {
				// TODO should be listen,when cert resfresh,should be resfresh
			},
			// a new leader has been elected
			OnNewLeader: func(identity string) {
			},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	leaderelection.RunOrDie(ctx, leaderElectionConfig)
	return nil
}
