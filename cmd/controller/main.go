package main

import (
	"flag"
	"os"

	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

var log = logf.Log.WithName("k8s-resource-protection")

func main() {
	var debug bool

	flag.BoolVar(&debug, "debug", false, "Set debug logging")
	flag.Parse()

	logf.SetLogger(zap.Logger(false))
	entryLog := log.WithName("entrypoint")

	entryLog.Info("setting up manager")
	mgr, err := manager.New(config.GetConfigOrDie(), manager.Options{})
	if err != nil {
		entryLog.Error(err, "unable to set up overall controller manager")
		os.Exit(1)
	}

	entryLog.Info("setting up webhook server")
	hookServer := mgr.GetWebhookServer()

	entryLog.Info("registering webhooks to the webhook server")
	hookServer.Register("/allowed-operations", &webhook.Admission{
		Handler: loggingWebhookHandler{
			Handler: &allowedOperationsValidator{},
			Log:     log.WithName("handlers"),
			Debug:   debug,
		},
	})

	entryLog.Info("starting manager")
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		entryLog.Error(err, "unable to run manager")
		os.Exit(1)
	}
}
