package cmd

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alphabill-org/alphabill/internal/network/protocol/genesis"
	rootgenesis "github.com/alphabill-org/alphabill/internal/rootchain/genesis"
	"github.com/alphabill-org/alphabill/internal/testutils/net"
	testsig "github.com/alphabill-org/alphabill/internal/testutils/sig"
	testtime "github.com/alphabill-org/alphabill/internal/testutils/time"
	"github.com/alphabill-org/alphabill/internal/util"
	"github.com/stretchr/testify/require"
)

func TestVD_UseClientForTx(t *testing.T) {
	homeDirVD := setupTestHomeDir(t, "vd")
	keysFileLocation := filepath.Join(homeDirVD, defaultKeysFileName)
	nodeGenesisFileLocation := filepath.Join(homeDirVD, nodeGenesisFileName)
	partitionGenesisFileLocation := filepath.Join(homeDirVD, "partition-genesis.json")
	testtime.MustRunInTime(t, 20*time.Second, func() {
		freePort, err := net.GetFreePort()
		require.NoError(t, err)
		dialAddr := fmt.Sprintf("localhost:%v", freePort)

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
			fmt.Println("Starting VD node")
			cmd := New()
			args := "vd --home " + homeDirVD + " -g " + partitionGenesisFileLocation + " -k " + keysFileLocation + " --server-address " + dialAddr
			cmd.baseCmd.SetArgs(strings.Split(args, " "))

			err := cmd.addAndExecuteCommand(ctx)
			require.ErrorIs(t, err, context.Canceled)
			appStoppedWg.Done()
		}()

		err = sendTxWithClient(ctx, dialAddr)
		// as the rootchain is not running the partition node never gets past the initializing status
		require.EqualError(t, err, `failed to get current round number: rpc error: code = Unavailable desc = invalid state: partition node status is "initializing"`)

		// Close the app
		ctxCancel()
		// Wait for test asserts to be completed
		appStoppedWg.Wait()
	})
}

func sendTxWithClient(ctx context.Context, dialAddr string) error {
	cmd := New()
	args := "vd-client register --hash " + "0x67588D4D37BF6F4D6C63CE4BDA38DA2B869012B1BC131DB07AA1D2B5BFD810DD" + " -u " + dialAddr + " --wait"
	cmd.baseCmd.SetArgs(strings.Split(args, " "))
	return cmd.addAndExecuteCommand(ctx)
}
