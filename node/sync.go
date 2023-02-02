package node

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/ngoduongkha/go-ethereum-cloner/database"
)

func (n *Node) sync(ctx context.Context) error {
	n.doSync()

	ticker := time.NewTicker(10 * time.Second)

	for {
		select {
		case <-ticker.C:
			n.doSync()

		case <-ctx.Done():
			ticker.Stop()
		}
	}
}

func (n *Node) doSync() {
	for _, peer := range n.knownPeers {
		if n.info.IP == peer.IP && n.info.Port == peer.Port {
			continue
		}

		if peer.IP == "" {
			continue
		}

		fmt.Printf("Searching for new Peers and their Blocks and Peers: '%s'\n", peer.TcpAddress())

		// Step 1: Query peer status to get the latest block hash
		status, err := queryPeerStatus(peer)
		if err != nil {
			fmt.Printf("ERROR: %s\n", err)
			fmt.Printf("Peer '%s' was removed from KnownPeers\n", peer.TcpAddress())

			n.RemovePeer(peer)

			continue
		}

		// Step 2: Join the peer to our known peers
		err = n.joinKnownPeers(peer)
		if err != nil {
			fmt.Printf("ERROR: %s\n", err)
			continue
		}

		// Step 3: Sync the peer's blocks
		err = n.syncBlocks(peer, status)
		if err != nil {
			fmt.Printf("ERROR: %s\n", err)
			continue
		}

		// Step 4: Sync the peer's known peers
		err = n.syncKnownPeers(status)
		if err != nil {
			fmt.Printf("ERROR: %s\n", err)
			continue
		}

		// Step 5: Sync the peer's pending transactions
		err = n.syncPendingTXs(peer, status.PendingTXs)
		if err != nil {
			fmt.Printf("ERROR: %s\n", err)
			continue
		}
	}
}

func (n *Node) syncBlocks(peer PeerNode, status StatusResponse) error {
	localBlockNumber := n.state.LatestBlock().Header.Number

	// If the peer has no blocks, ignore it
	if status.Hash.IsEmpty() {
		return nil
	}

	// If the peer has fewer blocks than us, ignore it
	if status.Number < localBlockNumber {
		return nil
	}

	// If it's the genesis block, and we already synced it, ignore it
	if status.Number == 0 && !n.state.LatestBlockHash().IsEmpty() {
		return nil
	}

	// Display found 1 new block if we sync the genesis block 0
	newBlocksCount := status.Number - localBlockNumber
	if localBlockNumber == 0 && status.Number == 0 {
		newBlocksCount = 1
	}
	fmt.Printf("Found %d new blocks from Peer %s\n", newBlocksCount, peer.TcpAddress())

	blocks, err := fetchBlocksFromPeer(peer, n.state.LatestBlockHash())
	if err != nil {
		return err
	}

	for _, block := range blocks {
		err = n.addBlock(block)
		if err != nil {
			return err
		}

		n.newSyncedBlocks <- block
	}

	return nil
}

func (n *Node) syncKnownPeers(status StatusResponse) error {
	for _, statusPeer := range status.KnownPeers {
		if !n.IsKnownPeer(statusPeer) {
			fmt.Printf("Found new Peer %s\n", statusPeer.TcpAddress())

			n.AddPeer(statusPeer)
		}
	}

	return nil
}

func (n *Node) syncPendingTXs(peer PeerNode, txs []database.SignedTx) error {
	for _, tx := range txs {
		err := n.AddPendingTX(tx, peer)
		if err != nil {
			return err
		}
	}

	return nil
}
 
func (n *Node) joinKnownPeers(peer PeerNode) error {
	if peer.connected {
		return nil
	}

	peerUrl := fmt.Sprintf(
		"%s://%s%s?%s=%s&%s=%d&%s=%s",
		peer.ApiProtocol(),
		peer.TcpAddress(),
		endpointAddPeer,
		endpointAddPeerQueryKeyIP,
		n.info.IP,
		endpointAddPeerQueryKeyPort,
		n.info.Port,
		endpointAddPeerQueryKeyMiner,
		n.info.Account.String(),
	)

	res, err := http.Get(peerUrl)
	if err != nil {
		return err
	}

	addPeerResponse := AddPeerResponse{}
	err = readResponse(res, &addPeerResponse)
	if err != nil {
		return err
	}
	if addPeerResponse.Error != "" {
		return fmt.Errorf(addPeerResponse.Error)
	}

	knownPeer := n.knownPeers[peer.TcpAddress()]
	knownPeer.connected = addPeerResponse.Success

	n.AddPeer(knownPeer)

	if !addPeerResponse.Success {
		return fmt.Errorf("unable to join KnownPeers of '%s'", peer.TcpAddress())
	}

	return nil
}

func queryPeerStatus(peer PeerNode) (StatusResponse, error) {
	url := fmt.Sprintf("%s://%s%s", peer.ApiProtocol(), peer.TcpAddress(), endpointStatus)
	res, err := http.Get(url)
	if err != nil {
		return StatusResponse{}, err
	}

	statusResponse := StatusResponse{}
	err = readResponse(res, &statusResponse)
	if err != nil {
		return StatusResponse{}, err
	}

	return statusResponse, nil
}

func fetchBlocksFromPeer(peer PeerNode, fromBlock database.Hash) ([]database.Block, error) {
	fmt.Printf("Importing blocks from Peer %s...\n", peer.TcpAddress())

	url := fmt.Sprintf(
		"%s://%s%s?%s=%s",
		peer.ApiProtocol(),
		peer.TcpAddress(),
		endpointSync,
		endpointSyncQueryKeyFromBlock,
		fromBlock.Hex(),
	)

	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	syncResponse := SyncResponse{}
	err = readResponse(res, &syncResponse)
	if err != nil {
		return nil, err
	}

	return syncResponse.Blocks, nil
}
