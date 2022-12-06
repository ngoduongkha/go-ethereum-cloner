package database

import (
	"encoding/json"
)

const genesisJson = `
{
  "genesis_time": "2022-12-04T00:00:00.000000000Z",
  "chain_id": "ethereum-cloner",
  "balances": {
    "kha": 1000000
  }
}`

type genesis struct {
	Balances map[Account]uint `json:"balances"`
}

func loadGenesis() (genesis, error) {
	var loadedGenesis genesis
	err := json.Unmarshal([]byte(genesisJson), &loadedGenesis)
	if err != nil {
		return genesis{}, err
	}

	return loadedGenesis, nil
}
