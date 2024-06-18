package types

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/suite"
)

type SuiteStorage struct {
	suite.Suite
}

func TestSuiteStorage(t *testing.T) {
	suite.Run(t, new(SuiteStorage))
}

func (s *SuiteStorage) TestStorageString() {
	storage := Storage{NewStateFromEthHashes(common.BytesToHash([]byte("key")), common.BytesToHash([]byte("value")))}
	str := "key:\"0x00000000000000000000000000000000000000000000000000000000006b6579\" value:\"0x00000000000000000000000000000000000000000000000000000076616c7565\" \n"
	s.Equal(str, storage.String())
}

func (s *SuiteStorage) TestStorageValidate() {
	testCases := []struct {
		name     string
		storage  Storage
		wantPass bool
	}{
		{
			name: "valid storage",
			storage: Storage{
				NewStateFromEthHashes(common.BytesToHash([]byte{1, 2, 3}), common.BytesToHash([]byte{1, 2, 3})),
			},
			wantPass: true,
		},
		{
			name: "empty storage key bytes",
			storage: Storage{
				{Key: ""},
			},
			wantPass: false,
		},
		{
			name: "duplicated storage key",
			storage: Storage{
				{Key: common.BytesToHash([]byte{1, 2, 3}).String()},
				{Key: common.BytesToHash([]byte{1, 2, 3}).String()},
			},
			wantPass: false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		err := tc.storage.Validate()
		if tc.wantPass {
			s.NoError(err, tc.name)
		} else {
			s.Error(err, tc.name)
		}
	}
}

func (s *SuiteStorage) TestStorageCopy() {
	testCases := []struct {
		name    string
		storage Storage
	}{
		{
			"single storage",
			Storage{
				NewStateFromEthHashes(common.BytesToHash([]byte{1, 2, 3}), common.BytesToHash([]byte{1, 2, 3})),
			},
		},
		{
			"empty storage key value bytes",
			Storage{
				{Key: common.Hash{}.String(), Value: common.Hash{}.String()},
			},
		},
		{
			"empty storage",
			Storage{},
		},
	}

	for _, tc := range testCases {
		tc := tc
		s.Require().Equal(tc.storage, tc.storage.Copy(), tc.name)
	}
}
