package main

import (
	"flag"
	"fmt"
	"os"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	server "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	"github.com/malisettirammurthy/nginx-operator-autoscaler/controllers"
)

func main() {
	var metricsAddr string
	var healthAddr string
	flag.StringVar(&metricsAddr, "metrics-bind-address", "0", "The address the metric endpoint binds to.")
	flag.StringVar(&healthAddr, "health-probe-bind-address", ":8081", "The address the health probe endpoint binds to.")
	flag.Parse()

	// Logger
	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	// Scheme (we only need built-in apps/v1 for Deployment)
	scheme := runtime.NewScheme()
	_ = appsv1.AddToScheme(scheme)

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		Metrics:                server.Options{BindAddress: metricsAddr},
		HealthProbeBindAddress: healthAddr,
		LeaderElection:         false,
	})
	if err != nil {
		panic(fmt.Errorf("manager: %w", err))
	}

	// Reconciler
	if err := controllers.SetupNginxAutoscalerController(mgr); err != nil {
		panic(fmt.Errorf("setup controller: %w", err))
	}

	_ = mgr.AddHealthzCheck("ping", healthz.Ping)
	_ = mgr.AddReadyzCheck("ping", healthz.Ping)

	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		fmt.Fprintln(os.Stderr, "manager stopped:", err)
		os.Exit(1)
	}
}
