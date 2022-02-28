package root

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCreateUnicityTree(t *testing.T) {
	sysIds := []*SystemId{{systemId: []byte{1}}, {systemId: []byte{2}}, {systemId: []byte{3}}}
	sysDescriptors := make(map[string]*SystemDescriptionRecord)
	sysInputs := make(map[string]*SystemInputRecord)
	for _, sysId := range sysIds {
		sysDescriptors[sysId.hashKey()] = &SystemDescriptionRecord{data: []byte{4}}
		sysInputs[sysId.hashKey()] = &SystemInputRecord{
			prevRootHash: []byte{5},
			rootHash:     []byte{6},
			summaryValue: []byte{7},
		}
	}

	ut := CreateUnicityTree(sysIds, sysDescriptors, sysInputs)
	require.NotNil(t, ut)
}
