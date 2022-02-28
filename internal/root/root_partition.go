package root

import (
	"bytes"
	"crypto/sha256"
	"github.com/lazyledger/smt"
)

type (
	State struct {
		roundNumber        uint64
		trustBase          []byte                              // uncity trust base
		systemIds          []*SystemId                         // set of system identifiers
		rootHashes         map[string][]byte                   // last certified root hashes of the registered transaction systems (systemId -> hash)
		systemDescriptions map[string]*SystemDescriptionRecord // system descriptions for all registered transaction systems
	}

	SystemDescriptionRecord struct {
		// spec 2.3 System Decription Record
		data []byte
	}

	SystemInputRecord struct {
		// spec 3.2 System Input Record
		prevRootHash []byte // previously certified root hash
		rootHash     []byte // root hash to be certified
		summaryValue []byte // summary value to be certified of type
	}

	SystemId struct {
		systemId []byte
	}
)

func CreateUnicityTree(systemIds []*SystemId, systemDescriptions map[string]*SystemDescriptionRecord, systemInputRecords map[string]*SystemInputRecord) *smt.SparseMerkleTree {
	sm := smt.NewSimpleMap()
	tree := smt.NewSparseMerkleTree(sm, sha256.New())
	hasher := sha256.New()
	for _, sysId := range systemIds {
		sysIdKey := sysId.hashKey()
		sysDesc := systemDescriptions[sysIdKey]
		sysInput := systemInputRecords[sysIdKey]
		hasher.Write(sysDesc.serialize())
		hasher.Write(sysInput.serialize())
		val := hasher.Sum(nil)
		hasher.Reset()
		_, _ = tree.Update(sysId.systemId, val)
	}
	return tree
}

func (sid *SystemId) hashKey() string {
	return string(sid.systemId)
}

func (sir *SystemInputRecord) serialize() []byte {
	b := bytes.Buffer{}
	b.Write(sir.prevRootHash)
	b.Write(sir.rootHash)
	b.Write(sir.summaryValue)
	return b.Bytes()
}

func (sdr *SystemDescriptionRecord) serialize() []byte {
	b := bytes.Buffer{}
	b.Write(sdr.data)
	return b.Bytes()
}
