package node

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ngoduongkha/go-ethereum-cloner/database"
	"github.com/ngoduongkha/go-ethereum-cloner/wallet"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

type BalancesResponse struct {
	Hash     database.Hash           `json:"block_hash"`
	Balances map[common.Address]uint `json:"balances"`
}

type AddTxRequest struct {
	From    string `json:"from"`
	FromPwd string `json:"from_pwd"`
	To      string `json:"to"`
	Value   uint   `json:"value"`
	Data    string `json:"data"`
}

type AddTxResponse struct {
	Success bool `json:"success"`
}

type StatusResponse struct {
	Hash       database.Hash       `json:"block_hash"`
	Number     uint64              `json:"block_number"`
	KnownPeers map[string]PeerNode `json:"peers_known"`
	PendingTXs []database.SignedTx `json:"pending_txs"`
	Account    common.Address      `json:"account"`
}

type NodeInfo struct {
	Nodes      []PeerNode          `json:"nodes"`
	Blocks     []database.Block    `json:"blocks"`
	PendingTXs []database.SignedTx `json:"pending_txs"`
}

type SyncResponse struct {
	Blocks []database.Block `json:"blocks"`
}

type AddPeerResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

func listBalancesHandler(w http.ResponseWriter, state *database.State) {
	enableCors(&w)

	writeResponse(w, BalancesResponse{state.LatestBlockHash(), state.Balances})
}

func addTxHandler(w http.ResponseWriter, r *http.Request, node *Node) {
	req := AddTxRequest{}
	err := readRequest(r, &req)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	from := database.NewAccount(req.From)

	if from.String() == common.HexToAddress("").String() {
		writeErrorResponse(w, fmt.Errorf("%s is an invalid 'from' sender", from.String()))
		return
	}

	if req.FromPwd == "" {
		writeErrorResponse(w, fmt.Errorf("password to decrypt the %s account is required. 'from_pwd' is empty", from.String()))
		return
	}

	nonce := node.state.GetNextAccountNonce(from)
	tx := database.NewTx(from, database.NewAccount(req.To), req.Value, nonce, req.Data)

	signedTx, err := wallet.SignTxWithKeystoreAccount(tx, from, req.FromPwd, wallet.GetKeystoreDirPath(node.dataDir))
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	err = node.AddPendingTX(signedTx, node.info)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeResponse(w, AddTxResponse{Success: true})
}

func nodeInfoHandler(w http.ResponseWriter, node *Node) {
	enableCors(&w)

	blocks, err := node.state.GetBlocks()
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	res := NodeInfo{
		Nodes:      node.KnownPeers(),
		PendingTXs: node.getPendingTXsAsArray(),
		Blocks:     blocks,
	}

	writeResponse(w, res)
}

func listBlockHashesHandler(w http.ResponseWriter, state *database.State) {
	enableCors(&w)

	hashes, err := state.GetBlocks()
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeResponse(w, hashes)
}

func statusHandler(w http.ResponseWriter, node *Node) {
	enableCors(&w)

	res := StatusResponse{
		Hash:       node.state.LatestBlockHash(),
		Number:     node.state.LatestBlock().Header.Number,
		KnownPeers: node.knownPeers,
		PendingTXs: node.getPendingTXsAsArray(),
		Account:    database.NewAccount(node.info.Account.String()),
	}

	writeResponse(w, res)
}

func syncHandler(w http.ResponseWriter, r *http.Request, node *Node) {
	reqHash := r.URL.Query().Get(endpointSyncQueryKeyFromBlock)

	hash := database.Hash{}
	err := hash.UnmarshalText([]byte(reqHash))
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	blocks, err := database.GetBlocksAfter(hash, node.dataDir)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeResponse(w, SyncResponse{Blocks: blocks})
}

func addPeerHandler(w http.ResponseWriter, r *http.Request, node *Node) {
	peerIP := r.URL.Query().Get(endpointAddPeerQueryKeyIP)
	peerPortRaw := r.URL.Query().Get(endpointAddPeerQueryKeyPort)
	minerRaw := r.URL.Query().Get(endpointAddPeerQueryKeyMiner)

	peerPort, err := strconv.ParseUint(peerPortRaw, 10, 32)
	if err != nil {
		writeResponse(w, AddPeerResponse{false, err.Error()})
		return
	}

	peer := NewPeerNode(peerIP, peerPort, false, database.NewAccount(minerRaw), true)

	node.AddPeer(peer)

	fmt.Printf("Peer '%s' was added into KnownPeers\n", peer.TcpAddress())

	writeResponse(w, AddPeerResponse{true, ""})
}

func blockByNumberOrHash(w http.ResponseWriter, r *http.Request, node *Node) {
	enableCors(&w)

	errorParamsRequired := errors.New("height or hash param is required")

	params := strings.Split(r.URL.Path, "/")[1:]
	if len(params) < 2 {
		writeErrorResponse(w, errorParamsRequired)
		return
	}

	p := strings.TrimSpace(params[1])
	if len(p) == 0 {
		writeErrorResponse(w, errorParamsRequired)
		return
	}
	hsh := ""
	height, err := strconv.ParseUint(p, 10, 64)
	if err != nil {
		hsh = p
	}

	block, err := database.GetBlockByHeightOrHash(node.state, height, hsh, node.dataDir)
	if err != nil {
		writeErrorResponse(w, err)
		return
	}

	writeResponse(w, block)
}

func mempoolViewer(w http.ResponseWriter, txs map[string]database.SignedTx) {
	enableCors(&w)

	writeResponse(w, txs)
}
