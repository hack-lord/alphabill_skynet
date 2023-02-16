package twb

import (
	"encoding/json"
	"fmt"

	"github.com/alphabill-org/alphabill/internal/block"
	"github.com/alphabill-org/alphabill/internal/txsystem"
)

type (
	Storage interface {
		Close() error
		GetBlockNumber() (uint64, error)
		SetBlockNumber(blockNumber uint64) error

		SaveTokenTypeCreator(id TokenTypeID, kind Kind, creator PubKey) error
		SaveTokenType(data *TokenUnitType, proof *Proof) error
		GetTokenType(id TokenTypeID) (*TokenUnitType, error)
		QueryTokenType(kind Kind, creator PubKey, startKey TokenTypeID, count int) ([]*TokenUnitType, TokenTypeID, error)

		SaveToken(data *TokenUnit, proof *Proof) error
		GetToken(id TokenID) (*TokenUnit, error)
		QueryTokens(kind Kind, owner Predicate, startKey TokenID, count int) ([]*TokenUnit, TokenID, error)
	}
)

type (
	TokenUnitType struct {
		// common
		ID                       TokenTypeID
		ParentTypeID             TokenTypeID
		Symbol                   string
		SubTypeCreationPredicate Predicate
		TokenCreationPredicate   Predicate
		InvariantPredicate       Predicate
		// fungible only
		DecimalPlaces uint32
		// nft only
		NftDataUpdatePredicate Predicate
		// meta
		Kind   Kind
		TxHash []byte
	}

	TokenUnit struct {
		// common
		ID     TokenID
		Symbol string
		TypeID TokenTypeID
		Owner  Predicate
		// fungible only
		Amount   uint64
		Decimals uint32
		// nft only
		NftURI                 string
		NftData                []byte
		NftDataUpdatePredicate Predicate
		// meta
		Kind   Kind
		TxHash []byte
	}

	TokenID     []byte
	TokenTypeID []byte
	Kind        byte

	Proof struct {
		BlockNumber uint64                `json:"blockNumber"`
		Tx          *txsystem.Transaction `json:"tx"`
		Proof       *block.BlockProof     `json:"proof"`
	}

	Predicate []byte
	PubKey    []byte
)

const (
	Any Kind = 1 << iota
	Fungible
	NonFungible
)

func (kind Kind) String() string {
	switch kind {
	case Any:
		return "all"
	case Fungible:
		return "fungible"
	case NonFungible:
		return "nft"
	}
	return "unknown"
}

func strToTokenKind(s string) (Kind, error) {
	switch s {
	case "all", "":
		return Any, nil
	case "fungible":
		return Fungible, nil
	case "nft":
		return NonFungible, nil
	}
	return Any, fmt.Errorf("%q is not valid token kind", s)
}

func (tu TokenUnit) MarshalJSON() ([]byte, error) {
	data := map[string]any{
		"id":     tu.ID,
		"symbol": tu.Symbol,
		"typeId": tu.TypeID,
		"owner":  tu.Owner,
		"txHash": tu.TxHash,
		"kind":   tu.Kind,
	}
	switch tu.Kind {
	case NonFungible:
		data["nftUri"] = tu.NftURI
		data["nftData"] = tu.NftData
		data["nftDataUpdatePredicate"] = tu.NftDataUpdatePredicate
	case Fungible:
		data["amount"] = tu.Amount
		data["decimals"] = tu.Decimals
	default:
		return nil, fmt.Errorf("unsupported token kind %d", tu.Kind)
	}

	return json.Marshal(data)
}

func (tt TokenUnitType) MarshalJSON() ([]byte, error) {
	data := map[string]any{
		"id":                       tt.ID,
		"parentTypeId":             tt.ParentTypeID,
		"symbol":                   tt.Symbol,
		"subTypeCreationPredicate": tt.SubTypeCreationPredicate,
		"tokenCreationPredicate":   tt.TokenCreationPredicate,
		"invariantPredicate":       tt.InvariantPredicate,
		"txHash":                   tt.TxHash,
		"kind":                     tt.Kind,
	}
	switch tt.Kind {
	case NonFungible:
		data["nftDataUpdatePredicate"] = tt.NftDataUpdatePredicate
	case Fungible:
		data["decimalPlaces"] = tt.DecimalPlaces
	default:
		return nil, fmt.Errorf("unsupported token kind %d", tt.Kind)
	}

	return json.Marshal(data)
}
