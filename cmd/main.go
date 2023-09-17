/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"flag"
	"net/http"
	"os"
	"time"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/tools/leaderelection/resourcelock"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	cgrecord "k8s.io/client-go/tools/record"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	infrastructurev1alpha1 "github.com/vpsie/cluster-api-provider-vpsie/api/v1alpha1"
	"github.com/vpsie/cluster-api-provider-vpsie/internal/controller"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(infrastructurev1alpha1.AddToScheme(scheme))
	utilruntime.Must(clusterv1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var reconcileTimeout time.Duration
	var leaderElectionLeaseDuration time.Duration
	var leaderElectionRenewDeadline time.Duration
	var leaderElectionRetryPeriod time.Duration
	var syncPeriod time.Duration
	var leaderElectionNamespace string
	var watchNamespace string
	var profilerAddress string
	var webhookPort int

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":9440", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.DurationVar(&reconcileTimeout, "reconcile-timeout", 10*time.Minute, "The maximum duration a reconcile loop can run.")
	flag.DurationVar(&leaderElectionLeaseDuration, "leader-election-lease-duration", 15*time.Second, "The duration that non-leader candidates will wait to force acquire leadership.")
	flag.DurationVar(&leaderElectionRenewDeadline, "leader-election-renew-deadline", 10*time.Second, "The duration that the acting leader will retry refreshing leadership before giving up.")
	flag.DurationVar(&leaderElectionRetryPeriod, "leader-election-retry-period", 2*time.Second, "The duration the LeaderElector clients should wait between tries of actions.")
	flag.DurationVar(&syncPeriod, "sync-period", 10*time.Minute, "The minimum interval at which watched resources are reconciled (e.g. 10m)")
	flag.StringVar(&leaderElectionNamespace, "leader-election-namespace", "", "Namespace used to create the ConfigMap that is used for holding the leader lock during leader election.")
	flag.StringVar(&watchNamespace, "watch-namespace", "", "Namespace that the controller watches to reconcile cluster-api objects. If unspecified, the controller watches for cluster-api objects across all namespaces.")
	flag.StringVar(&profilerAddress, "profiler-address", "", "Bind address to expose the pprof profiler (e.g. localhost:6060)")
	flag.IntVar(&webhookPort, "webhook-port", 9443, "Webhook Server port")

	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))
	ctx := ctrl.SetupSignalHandler()

	if watchNamespace != "" {
		setupLog.Info("Watching cluster-api objects only in namespace for reconciliation", "namespace", watchNamespace)
	}

	if profilerAddress != "" {
		setupLog.Info("Profiler listening for requests", "profiler-address", profilerAddress)
		go func() {
			server := &http.Server{
				Addr: profilerAddress,

				// Timeouts
				ReadTimeout:       60 * time.Second,
				ReadHeaderTimeout: 60 * time.Second,
				WriteTimeout:      60 * time.Second,
				IdleTimeout:       60 * time.Second,
			}
			err := server.ListenAndServe()
			if err != nil {
				setupLog.Error(err, "listen and serve error")
			}
		}()
	}

	// Machine and cluster operations can create enough events to trigger the event recorder spam filter
	// Setting the burst size higher ensures all events will be recorded and submitted to the API
	broadcaster := cgrecord.NewBroadcasterWithCorrelatorOptions(cgrecord.CorrelatorOptions{
		BurstSize: 100,
	})

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   webhookPort,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "5ba55f9f.cluster.x-k8s.io",

		LeaderElectionNamespace:    leaderElectionNamespace,
		LeaseDuration:              &leaderElectionLeaseDuration,
		RenewDeadline:              &leaderElectionRenewDeadline,
		RetryPeriod:                &leaderElectionRetryPeriod,
		LeaderElectionResourceLock: resourcelock.LeasesResourceLock,
		SyncPeriod:                 &syncPeriod,
		EventBroadcaster:           broadcaster,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager ")
		os.Exit(1)
	}

	if err = (&controller.VpsieClusterReconciler{
		Client:           mgr.GetClient(),
		Scheme:           mgr.GetScheme(),
		ReconcileTimeout: reconcileTimeout,
	}).SetupWithManager(ctx, mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "VpsieCluster")
		os.Exit(1)
	}
	if err = (&controller.VpsieMachineReconciler{
		Client:           mgr.GetClient(),
		Scheme:           mgr.GetScheme(),
		ReconcileTimeout: reconcileTimeout,
	}).SetupWithManager(ctx, mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "VpsieMachine")
		os.Exit(1)
	}

	// if err = (&infrastructurev1alpha1.VpsieCluster{}).SetupWebhookWithManager(mgr); err != nil {
	// 	setupLog.Error(err, "unable to create webhook", "webhook", "VpsieCluster")
	// 	os.Exit(1)
	// }
	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctx); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
