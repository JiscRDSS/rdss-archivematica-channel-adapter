package consumer

import (
	"context"
	"net/http"
	"os"
	"os/signal"

	log "github.com/Sirupsen/logrus"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/JiscRDSS/rdss-archivematica-channel-adapter/amclient"
	"github.com/JiscRDSS/rdss-archivematica-channel-adapter/broker"
	"github.com/JiscRDSS/rdss-archivematica-channel-adapter/broker/backend"
	"github.com/JiscRDSS/rdss-archivematica-channel-adapter/consumer"
	"github.com/JiscRDSS/rdss-archivematica-channel-adapter/s3"

	// Backend implementations
	_ "github.com/JiscRDSS/rdss-archivematica-channel-adapter/broker/backend/kinesis"

	// Serve runtime profiling data via HTTP
	_ "net/http/pprof"
)

var cmd = &cobra.Command{
	Use:   "consumer",
	Short: "Consumer server (RDSS » Archivematica)",
	Run: func(cmd *cobra.Command, args []string) {
		start()
	},
}

var logger log.FieldLogger

func Command(l log.FieldLogger) *cobra.Command {
	logger = l
	return cmd
}

func start() {
	logger.Infoln("Hello!")
	defer logger.Info("Bye!")

	go func() {
		logger.Errorln(http.ListenAndServe("0.0.0.0:6060", nil))
	}()

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	br, err := createBrokerClient()
	if err != nil {
		logger.Fatalln(err)
	}

	s3Client, err := createS3Client()
	if err != nil {
		logger.Fatalln(err)
	}

	depositDir := viper.GetString("consumer.archivematica_transfer_deposit_dir")
	depositFs := afero.NewBasePathFs(afero.NewOsFs(), depositDir)

	quit := make(chan struct{})
	go func() {
		c := consumer.MakeConsumer(ctx, logger, br, createAmClient(), s3Client, depositFs)
		c.Start()

		quit <- struct{}{}
	}()

	// Subscribe to SIGINT signals and wait
	stopChan := make(chan os.Signal)
	signal.Notify(stopChan, os.Interrupt)
	<-stopChan // Wait for SIGINT

	logger.Info("Shutting down server...")
	cancel()
	<-quit
}

func createBrokerClient() (*broker.Broker, error) {
	var (
		opts  = []backend.DialOpts{}
		qM    = viper.GetString("broker.queues.main")
		qI    = viper.GetString("broker.queues.invalid")
		qE    = viper.GetString("broker.queues.error")
		epKi  = viper.GetString("broker.kinesis.endpoint")
		epDb  = viper.GetString("broker.kinesis.dynamodb_endpoint")
		tlsKi = viper.GetString("broker.kinesis.tls")
	)
	if epKi != "" {
		opts = append(opts, backend.WithKeyValue("endpoint", epKi))
	}
	if epDb != "" {
		opts = append(opts, backend.WithKeyValue("dynamodb_endpoint", epDb))
	}
	if tlsKi != "" {
		opts = append(opts, backend.WithKeyValue("tls", tlsKi))
	}

	ba, err := backend.Dial("kinesis", opts...)
	if err != nil {
		return nil, err
	}
	return broker.New(ba, logger, &broker.Config{QueueMain: qM, QueueInvalid: qI, QueueError: qE})
}

func createAmClient() *amclient.Client {
	return amclient.NewClient(nil,
		viper.GetString("amclient.url"),
		viper.GetString("amclient.user"),
		viper.GetString("amclient.key"))
}

func createS3Client() (s3.ObjectStorage, error) {
	var (
		ep                 = viper.GetString("s3.endpoint")
		aKey               = viper.GetString("s3.access_key")
		sKey               = viper.GetString("s3.secret_key")
		region             = viper.GetString("s3.region")
		forcePathStyle     = viper.GetBool("s3.force_path_style")
		insecureSkipVerify = viper.GetBool("s3.insecure_skip_verify")
	)

	opts := []s3.ClientOpt{
		s3.SetForcePathStyle(forcePathStyle),
		s3.SetInsecureSkipVerify(insecureSkipVerify),
	}
	if ep != "" {
		opts = append(opts, s3.SetEndpoint(ep))
	}
	if aKey != "" && sKey != "" {
		opts = append(opts, s3.SetKeys(aKey, sKey))
	}
	if region != "" {
		opts = append(opts, s3.SetRegion(region))
	}
	return s3.New(opts...)
}