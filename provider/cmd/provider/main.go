// provider-gantry-cloud reconciles gantry-cloud networks and instances.
package main

import (
	"flag"
	"os"
	"time"

	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	"github.com/georgitinchev/gantry/provider/apis/v1alpha1"
	"github.com/georgitinchev/gantry/provider/internal/clients"
	"github.com/georgitinchev/gantry/provider/internal/controller"
)

func main() {
	var (
		debug    = flag.Bool("debug", false, "verbose logging")
		poll     = flag.Duration("poll", 30*time.Second, "how often to re-observe external resources")
		leader   = flag.Bool("leader-election", true, "use leader election")
		cloudURL = flag.String("cloud-url", os.Getenv("CLOUD_URL"), "gantry-cloud base URL")
	)
	flag.Parse()

	zl := zap.New(zap.UseDevMode(*debug))
	ctrl.SetLogger(zl)
	log := logging.NewLogrLogger(zl.WithName("provider-gantry-cloud"))

	token := os.Getenv("CLOUD_TOKEN")
	if *cloudURL == "" || token == "" {
		log.Info("CLOUD_URL and CLOUD_TOKEN are required")
		os.Exit(1)
	}

	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		panic(err)
	}
	if err := v1alpha1.AddToScheme(scheme); err != nil {
		panic(err)
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		LeaderElection:         *leader,
		LeaderElectionID:       "provider-gantry-cloud",
		HealthProbeBindAddress: ":8081",
		Metrics:                metricsserver.Options{BindAddress: ":8080"},
	})
	if err != nil {
		log.Info("cannot create manager", "error", err)
		os.Exit(1)
	}
	_ = mgr.AddHealthzCheck("healthz", healthz.Ping)
	_ = mgr.AddReadyzCheck("readyz", healthz.Ping)

	if err := controller.Setup(mgr, controller.Options{
		Cloud:        clients.New(*cloudURL, token),
		Logger:       log,
		PollInterval: *poll,
	}); err != nil {
		log.Info("cannot set up controllers", "error", err)
		os.Exit(1)
	}

	log.Info("starting", "cloudURL", *cloudURL, "poll", poll.String())
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		log.Info("manager stopped", "error", err)
		os.Exit(1)
	}
}
