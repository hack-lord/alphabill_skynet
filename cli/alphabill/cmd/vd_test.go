package cmd

import (
	"context"
	"fmt"
	"math/rand"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alphabill-org/alphabill/internal/network/protocol/genesis"
	rootgenesis "github.com/alphabill-org/alphabill/internal/rootchain/genesis"
	"github.com/alphabill-org/alphabill/internal/rpc/alphabill"
	"github.com/alphabill-org/alphabill/internal/testutils/net"
	testsig "github.com/alphabill-org/alphabill/internal/testutils/sig"
	testtime "github.com/alphabill-org/alphabill/internal/testutils/time"
	"github.com/alphabill-org/alphabill/internal/txsystem"
	"github.com/alphabill-org/alphabill/internal/util"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestRunVD(t *testing.T) {
	homeDirVD := setupTestHomeDir(t, "vd")
	keysFileLocation := filepath.Join(homeDirVD, defaultKeysFileName)
	nodeGenesisFileLocation := filepath.Join(homeDirVD, nodeGenesisFileName)
	partitionGenesisFileLocation := filepath.Join(homeDirVD, "partition-genesis.json")
	testtime.MustRunInTime(t, 5*time.Second, func() {
		nodeAddr := fmt.Sprintf("localhost:%d", net.GetFreeRandomPort(t))
		appStoppedWg := sync.WaitGroup{}
		ctx, ctxCancel := context.WithCancel(context.Background())

		// generate node genesis
		cmd := New()
		args := "vd-genesis --home " + homeDirVD + " -o " + nodeGenesisFileLocation + " -g -k " + keysFileLocation
		cmd.baseCmd.SetArgs(strings.Split(args, " "))
		require.NoError(t, cmd.addAndExecuteCommand(ctx))

		pn, err := util.ReadJsonFile(nodeGenesisFileLocation, &genesis.PartitionNode{})
		require.NoError(t, err)

		// use same keys for signing and communication encryption.
		rootSigner, verifier := testsig.CreateSignerAndVerifier(t)
		rootPubKeyBytes, err := verifier.MarshalPublicKey()
		require.NoError(t, err)
		pr, err := rootgenesis.NewPartitionRecordFromNodes([]*genesis.PartitionNode{pn})
		require.NoError(t, err)
		_, partitionGenesisFiles, err := rootgenesis.NewRootGenesis("test", rootSigner, rootPubKeyBytes, pr)
		require.NoError(t, err)

		err = util.WriteJsonFile(partitionGenesisFileLocation, partitionGenesisFiles[0])
		require.NoError(t, err)

		// start the node in background
		appStoppedWg.Add(1)
		go func() {

			cmd = New()
			args = "vd --home " + homeDirVD + " -g " + partitionGenesisFileLocation + " -k " + keysFileLocation + " --server-address " + nodeAddr
			cmd.baseCmd.SetArgs(strings.Split(args, " "))

			err = cmd.addAndExecuteCommand(ctx)
			require.ErrorIs(t, err, context.Canceled)
			appStoppedWg.Done()
		}()

		log.Info("Started vd node and dialing...")
		// Create the gRPC client
		conn, err := grpc.DialContext(ctx, nodeAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		require.NoError(t, err)
		defer conn.Close()
		rpcClient := alphabill.NewAlphabillServiceClient(conn)

		// Test
		id := uint256.NewInt(rand.Uint64()).Bytes32()
		tx := &txsystem.Transaction{
			UnitId:                id[:],
			TransactionAttributes: nil,
			ClientMetadata:        &txsystem.ClientMetadata{Timeout: 10},
			SystemId:              []byte{0, 0, 0, 1},
		}

		_, err = rpcClient.ProcessTransaction(ctx, tx, grpc.WaitForReady(true))
		// as the rootchain is not running the partition node never gets past the initializing status
		require.EqualError(t, err, `rpc error: code = Unavailable desc = invalid state: partition node status is "initializing"`)

		// Close the app
		ctxCancel()
		// Wait for test asserts to be completed
		appStoppedWg.Wait()
	})
}
