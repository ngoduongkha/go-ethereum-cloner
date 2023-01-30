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
	  "0x0eBa9c7AD60e5c0e45a709F93AF2A7a4BbFcd9c1": 1000000
	}
  }`

type Genesis struct {
	Balances map[common.Address]uint `json:"balances"`
	Symbol   string                  `json:"symbol"`
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
