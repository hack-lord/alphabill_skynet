package testutils

import (
	"fmt"

	abcrypto "github.com/alphabill-org/alphabill/crypto"
	"github.com/alphabill-org/alphabill/predicates/templates"
	moneytx "github.com/alphabill-org/alphabill/txsystem/money"
	"github.com/alphabill-org/alphabill/types"
	"github.com/alphabill-org/alphabill/wallet/account"
	"github.com/fxamacker/cbor/v2"
)

func CreateInitialBillTransferTx(accountKey *account.AccountKey, billID, fcrID types.UnitID, billValue uint64, timeout uint64, backlink []byte) (*types.TransactionOrder, error) {
	attr := &moneytx.TransferAttributes{
		NewBearer:   templates.NewP2pkh256BytesFromKey(accountKey.PubKey),
		TargetValue: billValue,
		Backlink:    backlink,
	}
	attrBytes, err := cbor.Marshal(attr)
	if err != nil {
		return nil, err
	}
	txo := &types.TransactionOrder{
		Payload: &types.Payload{
			SystemID:   []byte{0, 0, 0, 0},
			Type:       moneytx.PayloadTypeTransfer,
			UnitID:     billID,
			Attributes: attrBytes,
			ClientMetadata: &types.ClientMetadata{
				Timeout:           timeout,
				MaxTransactionFee: 1,
				FeeCreditRecordID: fcrID,
			},
		},
	}
	signer, err := abcrypto.NewInMemorySecp256K1SignerFromKey(accountKey.PrivKey)
	if err != nil {
		return nil, fmt.Errorf("creating signer from key: %w", err)
	}
	sigBytes, err := txo.PayloadBytes()
	if err != nil {
		return nil, fmt.Errorf("reading payload bytes: %w", err)
	}
	sigData, err := signer.SignBytes(sigBytes)
	if err != nil {
		return nil, fmt.Errorf("signing tx order: %w", err)
	}
	txo.OwnerProof = templates.NewP2pkh256SignatureBytes(sigData, accountKey.PubKey)
	return txo, nil
}
