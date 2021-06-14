package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	webui "github.com/jenkins-x-plugins/lighthouse-webui-plugin"
	"github.com/jenkins-x-plugins/lighthouse-webui-plugin/internal/kube"
	"github.com/jenkins-x-plugins/lighthouse-webui-plugin/internal/lighthouse"
	"github.com/jenkins-x-plugins/lighthouse-webui-plugin/internal/version"
	"github.com/jenkins-x-plugins/lighthouse-webui-plugin/web/handlers"

	lhclientset "github.com/jenkins-x/lighthouse/pkg/client/clientset/versioned"
	"github.com/sirupsen/logrus"
)

var (
	options struct {
		namespace             string
		resyncInterval        time.Duration
		lighthouseHMACKey     string
		keeperEndpoint        string
		keeperSyncInterval    time.Duration
		eventTraceURLTemplate string
		storeConfig           webui.StoreConfig
		kubeConfigPath        string
		listenAddr            string
		logLevel              string
		printVersion          bool
	}
)

func init() {
	flag.StringVar(&options.namespace, "namespace", "jx", "Name of the namespace with the lighthouse jobs")
	flag.DurationVar(&options.resyncInterval, "resync-interval", 1*time.Hour, "Resync interval between full re-list operations")
	flag.StringVar(&options.lighthouseHMACKey, "lighthouse-hmac-key", os.Getenv("LIGHTHOUSE_HMAC_KEY"), "HMAC key used by Lighthouse to sign the webhooks")
	flag.StringVar(&options.keeperEndpoint, "keeper-endpoint", "http://lighthouse-keeper.jx", "Endpoint of the Lighthouse Keeper service, to retrieve the Keeper state. Format: scheme://host:port")
	flag.DurationVar(&options.keeperSyncInterval, "keeper-sync-interval", 1*time.Minute, "Interval to poll the Lighthouse Keeper service for its state")
	flag.StringVar(&options.eventTraceURLTemplate, "event-trace-url-template", "", "Go template string used to build the event trace URL")
	flag.StringVar(&options.logLevel, "log-level", "INFO", "Log level - one of: trace, debug, info, warn(ing), error, fatal or panic")
	flag.StringVar(&options.storeConfig.DataPath, "store-data-path", "", "If non-empty, the events store will be persisted on disk in the directory")
	flag.IntVar(&options.storeConfig.MaxEvents, "store-max-events", 0, "If non-zero, the internal GC will ensure that no more than that many number of events will be stored/persisted")
	flag.DurationVar(&options.storeConfig.EventsMaxAge, "store-events-max-age", 0, "If non-zero, the internal GC will ensure to events older than this age (duration) will be removed from the store")
	flag.StringVar(&options.kubeConfigPath, "kubeconfig", kube.DefaultKubeConfigPath(), "Kubernetes Config Path. Default: KUBECONFIG env var value")
	flag.StringVar(&options.listenAddr, "listen-addr", ":8080", "Address on which the server will listen for incoming connections")
	flag.BoolVar(&options.printVersion, "version", false, "Print the version")
}

func main() {
	flag.Parse()

	if options.printVersion {
		fmt.Printf("Version %s - Revision %s - Date %s", version.Version, version.Revision, version.Date)
		return
	}

	ctx, cancelCtx := context.WithCancel(context.Background())
	defer cancelCtx()

	logger := logrus.New()
	logLevel, err := logrus.ParseLevel(options.logLevel)
	if err != nil {
		logger.WithField("logLevel", options.logLevel).Error("Failed to set log level")
	} else {
		logger.SetLevel(logLevel)
	}
	logger.WithField("logLevel", logLevel).WithField("version", version.Version).Info("Starting")

	kConfig, err := kube.NewConfig(options.kubeConfigPath)
	if err != nil {
		logger.WithError(err).Fatal("failed to create a Kubernetes config")
	}
	lhClient, err := lhclientset.NewForConfig(kConfig)
	if err != nil {
		logger.WithError(err).Fatal("failed to create a Lighthouse client")
	}

	store, err := webui.NewStore(options.storeConfig, logger)
	if err != nil {
		logger.WithError(err).Fatal("failed to create a new store")
	}

	logger.WithField("endpoint", options.keeperEndpoint).WithField("syncInterval", options.keeperSyncInterval).Info("Starting Keeper Syncer")
	(&webui.KeeperSyncer{
		KeeperEndpoint: options.keeperEndpoint,
		SyncInterval:   options.keeperSyncInterval,
		Store:          store,
		Logger:         logger,
	}).Start(ctx)

	lighthouseHandler := &lighthouse.Handler{
		SecretToken: options.lighthouseHMACKey,
		Logger:      logger,
	}
	lighthouseHandler.RegisterWebhookHandler((&webui.EventHandler{
		Store:  store,
		Logger: logger,
	}).HandleWebhook)

	logger.WithField("namespace", options.namespace).WithField("resyncInterval", options.resyncInterval).Info("Starting Informer")
	(&webui.JobInformer{
		LHClient:       lhClient,
		Namespace:      options.namespace,
		ResyncInterval: options.resyncInterval,
		Store:          store,
		Logger:         logger,
	}).Start(ctx)

	handler, err := handlers.Router{
		Store:                 store,
		EventTraceURLTemplate: options.eventTraceURLTemplate,
		LighthouseJobClient:   lhClient.LighthouseV1alpha1().LighthouseJobs(options.namespace),
		LighthouseHandler:     lighthouseHandler,
		Logger:                logger,
	}.Handler()
	if err != nil {
		logger.WithError(err).Fatal("failed to initialize the HTTP handler")
	}
	httpServer := http.Server{
		Handler: handler,
		Addr:    options.listenAddr,
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		<-ctx.Done()
		shutdownCtx, cancelShutdownFunc := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelShutdownFunc()

		logger.Info("Shutting down the HTTP server...")
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			logger.Warn("Timed-out while waiting for the HTTP server to shutdown!")
		}

		logger.Info("Closing the store...")
		if err := store.Close(); err != nil {
			logger.WithError(err).Warn("failed to close the store")
		}

		wg.Done()
	}()
	go func() {
		logger.WithField("listenAddr", options.listenAddr).Info("Starting HTTP Server")
		err = httpServer.ListenAndServe()
		if !errors.Is(err, http.ErrServerClosed) {
			logger.WithError(err).Fatal("failed to start HTTP server")
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	sig := <-c

	logger.WithField("signal", sig).Info("Interrupted by user (or killed) !")
	cancelCtx()
	wg.Wait()

	logger.Info("This is the end, my friend")
}
