package database

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"sort"

	"github.com/ethereum/go-ethereum/common"
)

const TxFee = uint(50)

type State struct {
	Balances      map[common.Address]uint
	Account2Nonce map[common.Address]uint

	dbFile *os.File

	latestBlock     Block
	latestBlockHash Hash
	hasGenesisBlock bool

	miningDifficulty uint
	// position of block in file db
	HashCache   map[string]int64
	HeightCache map[uint64]int64
}

func NewStateFromDisk(dataDir string, miningDifficulty uint) (*State, error) {
	err := InitDataDirIfNotExists(dataDir, []byte(genesisJson))
	if err != nil {
		return nil, err
	}

	gen, err := loadGenesis(getGenesisJsonFilePath(dataDir))
	if err != nil {
		return nil, err
	}

	balances := make(map[common.Address]uint)
	for account, balance := range gen.Balances {
		balances[account] = balance
	}

	account2nonce := make(map[common.Address]uint)

	dbFilepath := getBlocksDbFilePath(dataDir)
	f, err := os.OpenFile(dbFilepath, os.O_APPEND|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(f)

	state := &State{balances, account2nonce, f, Block{}, Hash{}, false, miningDifficulty, map[string]int64{}, map[uint64]int64{}}

	// set file position
	filePos := int64(0)

	for scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return nil, err
		}

		blockFsJson := scanner.Bytes()

		if len(blockFsJson) == 0 {
			break
		}

		var blockFs BlockFS
		err = json.Unmarshal(blockFsJson, &blockFs)
		if err != nil {
			return nil, err
		}

		err = applyBlock(blockFs.Value, state)
		if err != nil {
			return nil, err
		}

		// set search caches
		state.HashCache[blockFs.Key.Hex()] = filePos
		state.HeightCache[blockFs.Value.Header.Number] = filePos
		filePos += int64(len(blockFsJson)) + 1

		state.latestBlock = blockFs.Value
		state.latestBlockHash = blockFs.Key
		state.hasGenesisBlock = true
	}

	return state, nil
}

func (s *State) GetForkedBlock(peerBlocks []Block) (Block, error) {
	blocks, err := s.GetBlocks()
	if err != nil {
		return Block{}, err
	}

	prev := Block{}
	for i, b := range blocks {
		if !reflect.DeepEqual(b, peerBlocks[i]) {
			if b.Header.Time < peerBlocks[i].Header.Time {
				return prev, nil
			}
		}
		prev = b
	}

	return Block{}, fmt.Errorf("no fork found")
}

func (s *State) RemoveBlocks(fromBlock Block) error {
	for !reflect.DeepEqual(s.latestBlock, fromBlock) {
		filePos, ok := s.HashCache[s.latestBlockHash.Hex()]
		if !ok {
			return fmt.Errorf("block not found")
		}

		for _, tx := range s.latestBlock.TXs {
			s.Balances[tx.From] += tx.Cost()
			s.Balances[tx.To] -= tx.Value
			s.Account2Nonce[tx.From]--
		}

		s.Balances[s.latestBlock.Header.Miner] -= BlockReward
		s.Balances[s.latestBlock.Header.Miner] -= uint(len(s.latestBlock.TXs)) * TxFee

		parent, err := GetBlockByHeightOrHashByFileName(s, 0, s.latestBlock.Header.Parent.Hex(), s.dbFile.Name())
		if err != nil {
			return err
		}

		s.latestBlock = parent.Value
		s.latestBlockHash = parent.Key
		delete(s.HashCache, s.latestBlockHash.Hex())
		delete(s.HeightCache, s.latestBlock.Header.Number)

		// truncate dbfile
		err = s.dbFile.Truncate(filePos)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *State) AddBlock(b Block) (Hash, error) {
	pendingState := s.Copy()

	err := applyBlock(b, &pendingState)
	if err != nil {
		return Hash{}, err
	}

	blockHash, err := b.Hash()
	if err != nil {
		return Hash{}, err
	}

	blockFs := BlockFS{blockHash, b}

	blockFsJson, err := json.Marshal(blockFs)
	if err != nil {
		return Hash{}, err
	}

	fmt.Printf("\nPersisting new Block to disk:\n")
	fmt.Printf("\t%s\n", blockFsJson)

	// get file pos for cache
	fs, _ := s.dbFile.Stat()
	filePos := fs.Size() + 1

	_, err = s.dbFile.Write(append(blockFsJson, '\n'))
	if err != nil {
		return Hash{}, err
	}

	// set search caches
	s.HashCache[blockFs.Key.Hex()] = filePos
	s.HeightCache[blockFs.Value.Header.Number] = filePos

	s.Balances = pendingState.Balances
	s.Account2Nonce = pendingState.Account2Nonce
	s.latestBlockHash = blockHash
	s.latestBlock = b
	s.hasGenesisBlock = true
	s.miningDifficulty = pendingState.miningDifficulty

	return blockHash, nil
}

func (s *State) NextBlockNumber() uint64 {
	if !s.hasGenesisBlock {
		return uint64(0)
	}

	return s.LatestBlock().Header.Number + 1
}

func (s *State) LatestBlock() Block {
	return s.latestBlock
}

func (s *State) LatestBlockHash() Hash {
	return s.latestBlockHash
}

func (s *State) GetNextAccountNonce(account common.Address) uint {
	return s.Account2Nonce[account] + 1
}

func (s *State) ChangeMiningDifficulty(newDifficulty uint) {
	s.miningDifficulty = newDifficulty
}

func (s *State) Copy() State {
	c := State{}
	c.hasGenesisBlock = s.hasGenesisBlock
	c.latestBlock = s.latestBlock
	c.latestBlockHash = s.latestBlockHash
	c.Balances = make(map[common.Address]uint)
	c.Account2Nonce = make(map[common.Address]uint)
	c.miningDifficulty = s.miningDifficulty

	for acc, balance := range s.Balances {
		c.Balances[acc] = balance
	}

	for acc, nonce := range s.Account2Nonce {
		c.Account2Nonce[acc] = nonce
	}

	return c
}

func (s *State) Close() error {
	return s.dbFile.Close()
}

func applyBlock(b Block, s *State) error {
	nextExpectedBlockNumber := s.latestBlock.Header.Number + 1

	if s.hasGenesisBlock && b.Header.Number != nextExpectedBlockNumber {
		return fmt.Errorf("next expected block must be '%d' not '%d'", nextExpectedBlockNumber, b.Header.Number)
	}

	if s.hasGenesisBlock && s.latestBlock.Header.Number > 0 && !reflect.DeepEqual(b.Header.Parent, s.latestBlockHash) {
		return fmt.Errorf("next block parent hash must be '%x' not '%x'", s.latestBlockHash, b.Header.Parent)
	}

	hash, err := b.Hash()
	if err != nil {
		return err
	}

	if !IsBlockHashValid(hash, s.miningDifficulty) {
		return fmt.Errorf("invalid block hash %x", hash)
	}

	err = applyTXs(b.TXs, s)
	if err != nil {
		return err
	}

	s.Balances[b.Header.Miner] += BlockReward
	s.Balances[b.Header.Miner] += uint(len(b.TXs)) * TxFee

	return nil
}

func applyTXs(txs []SignedTx, s *State) error {
	sort.Slice(txs, func(i, j int) bool {
		return txs[i].Time < txs[j].Time
	})

	for _, tx := range txs {
		err := ApplyTx(tx, s)
		if err != nil {
			return err
		}
	}

	return nil
}

func ApplyTx(tx SignedTx, s *State) error {
	err := ValidateTx(tx, s)
	if err != nil {
		return err
	}

	s.Balances[tx.From] -= tx.Cost()
	s.Balances[tx.To] += tx.Value

	s.Account2Nonce[tx.From] = tx.Nonce

	return nil
}

func ValidateTx(tx SignedTx, s *State) error {
	ok, err := tx.IsAuthentic()
	if err != nil {
		return err
	}

	if !ok {
		return fmt.Errorf("wrong TX. Sender '%s' is forged", tx.From.String())
	}

	expectedNonce := s.GetNextAccountNonce(tx.From)
	if tx.Nonce != expectedNonce {
		return fmt.Errorf("wrong TX. Sender '%s' next nonce must be '%d', not '%d'", tx.From.String(), expectedNonce, tx.Nonce)
	}

	if _, ok := s.Balances[tx.From]; !ok {
		s.Balances[tx.From] = 0
	}

	if tx.Cost() > s.Balances[tx.From] {
		return fmt.Errorf("wrong TX. Sender '%s' balance is %d ETH. Tx cost is %d ETH", tx.From.String(), s.Balances[tx.From], tx.Cost())
	}

	return nil
}

func (s *State) GetBlocks() ([]Block, error) {
	var blocks []Block

	_, err := s.dbFile.Seek(0, 0)
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(s.dbFile)

	for scanner.Scan() {
		var blockFs BlockFS
		err := json.Unmarshal(scanner.Bytes(), &blockFs)
		if err != nil {
			return nil, err
		}

		blocks = append(blocks, blockFs.Value)
	}

	return blocks, nil
}
