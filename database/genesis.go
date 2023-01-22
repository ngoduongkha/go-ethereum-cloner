package database

import (
	"encoding/json"
	"os"

	"github.com/ethereum/go-ethereum/common"
)

var genesisJson = `{
	"genesis_time": "2020-06-01T00:00:00.000000000Z",
	"chain_id": "Ethereum",
	"symbol": "ETH",
	"balances": {
	  "0x09eE50f2F37FcBA1845dE6FE5C762E83E65E755c": 1000000
	},
	"fork_tip_1": 35
  }`

type Genesis struct {
	Balances map[common.Address]uint `json:"balances"`
	Symbol   string                  `json:"symbol"`

	ForkTIP1 uint64 `json:"fork_tip_1"`
}

func loadGenesis(path string) (Genesis, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return Genesis{}, err
	}

	var loadedGenesis Genesis
	err = json.Unmarshal(content, &loadedGenesis)
	if err != nil {
		return Genesis{}, err
	}

	return loadedGenesis, nil
}

func writeGenesisToDisk(path string, genesis []byte) error {
	return os.WriteFile(path, genesis, 0o644)
}
