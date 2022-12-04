package database

import (
	"encoding/json"
	"os"
)

type genesis struct {
	Balances map[Account]uint `json:"balances"`
}

func loadGenesis(path string) (genesis, error) {
	jsonBlob, err := os.ReadFile(path)
	if err != nil {
		return genesis{}, err
	}

	var loadedGenesis genesis
	err = json.Unmarshal(jsonBlob, &loadedGenesis)
	if err != nil {
		return genesis{}, err
	}

	return loadedGenesis, nil
}
