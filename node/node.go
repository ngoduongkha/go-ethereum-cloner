package node

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ngoduongkha/go-ethereum-cloner/database"
)

const (
	DefaultIP        = "127.0.0.1"
	BootstrapPort    = 3000
	BootstrapAccount = "0xe153037747eadbDAA34a3D8c07dBd1F86dc7a17C"
)

const endpointStatus = "/node/status"

const (
	endpointSync                  = "/node/sync"
	endpointSyncQueryKeyFromBlock = "fromBlock"
)

const (
	endpointAddPeer              = "/node/peer"
	endpointAddPeerQueryKeyIP    = "ip"
	endpointAddPeerQueryKeyPort  = "port"
	endpointAddPeerQueryKeyMiner = "miner"
)

const (
	endpointBlockByNumberOrHash = "/block/"
	endpointMempoolViewer       = "/mempool/"
)

const (
	miningIntervalSeconds           = 1
	syncIntervalSeconds             = 4
	checkForkedStateIntervalSeconds = 10
	DefaultMiningDifficulty         = 2
)

type PeerNode struct {
	IP          string         `json:"ip"`
	Port        uint64         `json:"port"`
	IsBootstrap bool           `json:"is_bootstrap"`
	Account     common.Address `json:"account"`

	// Whenever my node already established connection, sync with this Peer
	connected bool
}

func (pn PeerNode) TcpAddress() string {
	return fmt.Sprintf("%s:%d", pn.IP, pn.Port)
}

func (pn PeerNode) ApiProtocol() string {
	return "http"
}

type Node struct {
	dataDir string
	info    PeerNode

	// The main blockchain state after all TXs from mined blocks were applied
	state *database.State

	// temporary pending state validating new incoming TXs but reset after the block is mined
	pendingState *database.State

	knownPeers      map[string]PeerNode
	pendingTXs      map[string]database.SignedTx
	archivedTXs     map[string]database.SignedTx
	newSyncedBlocks chan database.Block
	newPendingTXs   chan database.SignedTx

	// Number of zeroes the hash must start with to be considered valid. Default 3
	miningDifficulty uint
	isMining         bool
}

func New(dataDir string, ip string, port uint64, acc common.Address, bootstrap PeerNode, miningDifficulty uint) *Node {
	knownPeers := make(map[string]PeerNode)

	n := &Node{
		dataDir:          dataDir,
		info:             NewPeerNode(ip, port, false, acc, true),
		knownPeers:       knownPeers,
		pendingTXs:       make(map[string]database.SignedTx),
		archivedTXs:      make(map[string]database.SignedTx),
		newSyncedBlocks:  make(chan database.Block),
		newPendingTXs:    make(chan database.SignedTx, 10000),
		isMining:         false,
		miningDifficulty: miningDifficulty,
	}

	n.AddPeer(bootstrap)

	return n
}

func NewPeerNode(ip string, port uint64, isBootstrap bool, acc common.Address, connected bool) PeerNode {
	return PeerNode{ip, port, isBootstrap, acc, connected}
}

func (n *Node) Run(ctx context.Context) error {
	fmt.Printf("Listening on: %s:%d\n", n.info.IP, n.info.Port)

	state, err := database.NewStateFromDisk(n.dataDir, n.miningDifficulty)
	if err != nil {
		return err
	}
	defer func(state *database.State) {
		err := state.Close()
		if err != nil {
			fmt.Println("Error closing state:", err)
		}
	}(state)

	n.state = state

	pendingState := state.Copy()
	n.pendingState = &pendingState

	fmt.Println("Blockchain state:")
	fmt.Printf("	- height: %d\n", n.state.LatestBlock().Header.Number)
	fmt.Printf("	- hash: %s\n", n.state.LatestBlockHash().Hex())

	go func() {
		err := n.sync(ctx)
		if err != nil {
			fmt.Println("Error syncing:", err)
		}
	}()
	go func() {
		err := n.mine(ctx)
		if err != nil {
			fmt.Println("Error mining:", err)
		}
	}()

	return n.serveHttp(ctx)
}

func (n *Node) LatestBlockHash() database.Hash {
	return n.state.LatestBlockHash()
}

// Serve both HTTP and socketIO
func (n *Node) serveHttp(ctx context.Context) error {
	mux := http.NewServeMux()

	mux.HandleFunc("/balances/list", func(w http.ResponseWriter, r *http.Request) {
		listBalancesHandler(w, n.state)
	})

	mux.HandleFunc("/tx/add", func(w http.ResponseWriter, r *http.Request) {
		addTxHandler(w, r, n)
	})

	mux.HandleFunc("/node/info", func(w http.ResponseWriter, r *http.Request) {
		nodeInfoHandler(w, n)
	})

	// Get the list of block hashes
	mux.HandleFunc("/blocks/list", func(w http.ResponseWriter, r *http.Request) {
		listBlockHashesHandler(w, n.state)
	})

	mux.HandleFunc(endpointStatus, func(w http.ResponseWriter, r *http.Request) {
		statusHandler(w, n)
	})

	mux.HandleFunc(endpointSync, func(w http.ResponseWriter, r *http.Request) {
		syncHandler(w, r, n)
	})

	mux.HandleFunc(endpointAddPeer, func(w http.ResponseWriter, r *http.Request) {
		addPeerHandler(w, r, n)
	})

	mux.HandleFunc(endpointBlockByNumberOrHash, func(w http.ResponseWriter, r *http.Request) {
		blockByNumberOrHash(w, r, n)
	})

	mux.HandleFunc(endpointMempoolViewer, func(w http.ResponseWriter, r *http.Request) {
		mempoolViewer(w, n.pendingTXs)
	})

	server := &http.Server{Addr: fmt.Sprintf(":%d", n.info.Port), Handler: mux}

	go func() {
		<-ctx.Done()
		_ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(_ctx); err != nil {
			fmt.Println("Error shutting down server:", err)
		}
	}()

	return server.ListenAndServe()
}

func (n *Node) mine(ctx context.Context) error {
	var miningCtx context.Context
	var stopCurrentMining context.CancelFunc

	ticker := time.NewTicker(time.Second * miningIntervalSeconds)

	for {
		select {
		case <-ticker.C:
			go func() {
				if len(n.pendingTXs) > 0 && !n.isMining {
					n.isMining = true

					miningCtx, stopCurrentMining = context.WithCancel(ctx)
					err := n.minePendingTXs(miningCtx)
					if err != nil {
						fmt.Printf("ERROR: %s\n", err)
					}

					n.isMining = false
				}
			}()

		case block := <-n.newSyncedBlocks:
			if n.isMining {
				blockHash, _ := block.Hash()
				fmt.Printf("\nPeer mined next Block '%s' faster :(\n", blockHash.Hex())

				n.removeMinedPendingTXs(block)
				stopCurrentMining()
			}

		case <-ctx.Done():
			ticker.Stop()
			return nil
		}
	}
}

func (n *Node) minePendingTXs(ctx context.Context) error {
	blockToMine := NewPendingBlock(
		n.state.LatestBlockHash(),
		n.state.NextBlockNumber(),
		n.info.Account,
		n.getPendingTXsAsArray(),
	)

	minedBlock, err := Mine(ctx, blockToMine, n.miningDifficulty)
	if err != nil {
		return err
	}

	n.removeMinedPendingTXs(minedBlock)

	err = n.addBlock(minedBlock)
	if err != nil {
		return err
	}

	return nil
}

func (n *Node) removeMinedPendingTXs(block database.Block) {
	if len(block.TXs) > 0 && len(n.pendingTXs) > 0 {
		fmt.Println("Updating in-memory Pending TXs Pool:")
	}

	for _, tx := range block.TXs {
		txHash, _ := tx.Hash()
		if _, exists := n.pendingTXs[txHash.Hex()]; exists {
			fmt.Printf("\t-archiving mined TX: %s\n", txHash.Hex())

			n.archivedTXs[txHash.Hex()] = tx
			delete(n.pendingTXs, txHash.Hex())
		}
	}
}

func (n *Node) ChangeMiningDifficulty(newDifficulty uint) {
	n.miningDifficulty = newDifficulty
	n.state.ChangeMiningDifficulty(newDifficulty)
}

func (n *Node) AddPeer(peer PeerNode) {
	n.knownPeers[peer.TcpAddress()] = peer
}

func (n *Node) RemovePeer(peer PeerNode) {
	delete(n.knownPeers, peer.TcpAddress())
}

func (n *Node) IsKnownPeer(peer PeerNode) bool {
	if peer.IP == n.info.IP && peer.Port == n.info.Port {
		return true
	}

	_, isKnownPeer := n.knownPeers[peer.TcpAddress()]

	return isKnownPeer
}

func (n *Node) AddPendingTX(tx database.SignedTx, fromPeer PeerNode) error {
	txHash, err := tx.Hash()
	if err != nil {
		return err
	}

	txJson, err := json.Marshal(tx)
	if err != nil {
		return err
	}

	_, isAlreadyPending := n.pendingTXs[txHash.Hex()]
	_, isArchived := n.archivedTXs[txHash.Hex()]

	if !isAlreadyPending && !isArchived {
		err = n.validateTxBeforeAddingToMempool(tx)
		if err != nil {
			return err
		}

		fmt.Printf("Added Pending TX %s from Peer %s\n", txJson, fromPeer.TcpAddress())
		n.pendingTXs[txHash.Hex()] = tx
		n.newPendingTXs <- tx
	}

	return nil
}

func (n *Node) addBlock(block database.Block) error {
	_, err := n.state.AddBlock(block)
	if err != nil {
		return err
	}

	// Reset the pending state
	pendingState := n.state.Copy()
	n.pendingState = &pendingState

	return nil
}

// validateTxBeforeAddingToMempool ensures the TX is authentic, with correct nonce, and the sender has sufficient
// funds, so we waste PoW resources on TX we can tell in advance are wrong.
func (n *Node) validateTxBeforeAddingToMempool(tx database.SignedTx) error {
	return database.ApplyTx(tx, n.pendingState)
}

func (n *Node) getPendingTXsAsArray() []database.SignedTx {
	txs := make([]database.SignedTx, len(n.pendingTXs))

	i := 0
	for _, tx := range n.pendingTXs {
		txs[i] = tx
		i += 1
	}

	return txs
}

// Get known peers as an array
func (n *Node) KnownPeers() []PeerNode {
	peers := make([]PeerNode, len(n.knownPeers))

	i := 0
	for _, peer := range n.knownPeers {
		peers[i] = peer
		i += 1
	}

	return peers
}
