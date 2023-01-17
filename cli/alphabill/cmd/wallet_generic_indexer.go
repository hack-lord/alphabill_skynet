package cmd

import (
	"context"
	"fmt"
	"os"
	"path"
	"sync"
	"syscall"

	aberrors "github.com/alphabill-org/alphabill/internal/errors"
	"github.com/alphabill-org/alphabill/pkg/client"
	"github.com/alphabill-org/alphabill/pkg/wallet"
	backend "github.com/alphabill-org/alphabill/pkg/wallet/backend/generic_indexer"
	wlog "github.com/alphabill-org/alphabill/pkg/wallet/log"
	"github.com/spf13/cobra"
)

const (
	genericIndexerHomeDir = "generic-indexer"
)

type genericIndexerConfig struct {
	Base               *baseConfiguration
	AlphabillUrl       string
	ServerAddr         string
	DbFile             string
	LogLevel           string
	LogFile            string
	ListBillsPageLimit int
}

func (c *genericIndexerConfig) GetDbFile() (string, error) {
	if c.DbFile != "" {
		return c.DbFile, nil
	}
	indexerHomeDir := path.Join(c.Base.HomeDir, genericIndexerHomeDir)
	err := os.MkdirAll(indexerHomeDir, 0700) // -rwx------
	if err != nil {
		return "", err
	}
	return path.Join(indexerHomeDir, backend.BoltBillStoreFileName), nil
}

// newGenericIndexerCmd creates a new cobra command for the generic-indexer component.
func newGenericIndexerCmd(ctx context.Context, baseConfig *baseConfiguration) *cobra.Command {
	config := &genericIndexerConfig{Base: baseConfig}
	var walletCmd = &cobra.Command{
		Use:   "generic-indexer",
		Short: "starts generic indexer service",
		Long:  "starts generic indexer service, indexes all transactions by owner predicates, starts http server",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// initialize config so that baseConfig.HomeDir gets configured
			err := initializeConfig(cmd, baseConfig)
			if err != nil {
				return err
			}
			// init logger
			return initWalletLogger(&walletConfig{LogLevel: config.LogLevel, LogFile: config.LogFile})
		},
		Run: func(cmd *cobra.Command, args []string) {
			consoleWriter.Println("Error: must specify a subcommand")
		},
	}
	walletCmd.PersistentFlags().StringVar(&config.LogFile, logFileCmdName, "", fmt.Sprintf("log file path (default output to stderr)"))
	walletCmd.PersistentFlags().StringVar(&config.LogLevel, logLevelCmdName, "INFO", fmt.Sprintf("logging level (DEBUG, INFO, NOTICE, WARNING, ERROR)"))
	walletCmd.AddCommand(startGenericIndexerCmd(ctx, config))
	return walletCmd
}

func startGenericIndexerCmd(ctx context.Context, config *genericIndexerConfig) *cobra.Command {
	cmd := &cobra.Command{
		Use: "start",
		RunE: func(cmd *cobra.Command, args []string) error {
			return execGenericIndexerStartCmd(ctx, cmd, config)
		},
	}
	cmd.Flags().StringVarP(&config.AlphabillUrl, alphabillUriCmdName, "u", defaultAlphabillUri, "alphabill node url")
	cmd.Flags().StringVarP(&config.ServerAddr, serverAddrCmdName, "s", "localhost:9654", "server address")
	cmd.Flags().StringVarP(&config.DbFile, dbFileCmdName, "f", "", "path to the database file (default: $AB_HOME/generic-indexer/"+backend.BoltBillStoreFileName+")")
	cmd.Flags().IntVarP(&config.ListBillsPageLimit, listBillsPageLimit, "l", 100, "GET /list-bills request default/max limit size")
	return cmd
}

func execGenericIndexerStartCmd(ctx context.Context, _ *cobra.Command, config *genericIndexerConfig) error {
	abclient := client.New(client.AlphabillClientConfig{Uri: config.AlphabillUrl})
	dbFile, err := config.GetDbFile()
	if err != nil {
		return err
	}
	store, err := backend.NewBoltBillStore(dbFile)
	if err != nil {
		return err
	}
	bp := backend.NewBlockProcessor(store)
	w := wallet.New().SetBlockProcessor(bp).SetABClient(abclient).Build()

	service := backend.New(w, store)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		service.StartProcess(ctx)
		wg.Done()
	}()

	server := backend.NewHttpServer(config.ServerAddr, config.ListBillsPageLimit, service)
	err = server.Start()
	if err != nil {
		service.Shutdown()
		return aberrors.Wrap(err, "error starting wallet backend http server")
	}

	// listen for termination signal and shutdown the app
	hook := func(sig os.Signal) {
		wlog.Info("Received signal '", sig, "' shutting down application...")
		err := server.Shutdown(context.Background())
		if err != nil {
			wlog.Error("error shutting down server: ", err)
		}
		service.Shutdown()
	}
	listen(hook, syscall.SIGTERM, syscall.SIGKILL, syscall.SIGQUIT, syscall.SIGINT)

	wg.Wait() // wait for service shutdown to complete

	return nil
}
