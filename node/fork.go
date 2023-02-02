package node

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/ngoduongkha/go-ethereum-cloner/database"
)

// Check state is forked and remove forked blocks
func (n *Node) checkForkedState(ctx context.Context) error {
	n.doCheckForkedState()

	ticker := time.NewTicker(checkForkedStateIntervalSeconds * time.Second)

	for {
		select {
		case <-ticker.C:
			n.doCheckForkedState()

		case <-ctx.Done():
			ticker.Stop()
		}
	}
}

func (n *Node) doCheckForkedState() {
	for _, peer := range n.knownPeers {
		if n.info.IP == peer.IP && n.info.Port == peer.Port {
			continue
		}

		if peer.IP == "" {
			continue
		}

		fmt.Printf("Checking if State is Forked: '%s'\n", peer.TcpAddress())

		// Step 1: Query peer status to get the latest block hash
		status, err := queryPeerStatus(peer)
		if err != nil {
			fmt.Printf("ERROR: %s\n", err)
			fmt.Printf("Peer '%s' was removed from KnownPeers\n", peer.TcpAddress())

			n.RemovePeer(peer)

			continue
		}

		fmt.Println(111111)
		if status.Hash == n.state.LatestBlockHash() {
			continue
		}

		fmt.Println(22222)
		if status.Number <= n.state.LatestBlock().Header.Number {
			continue
		}

		// Step 2: query list blocks from peer
		peerBlocks, err := n.getBlocksFromPeer(peer)
		if err != nil {
			fmt.Printf("ERROR: %s\n", err)
			continue
		}

		fmt.Println(44444)

		// Step 3: find forked blocks
		forkedBlock, err := n.state.GetForkedBlock(peerBlocks)
		if err != nil {
			fmt.Printf("ERROR: %s\n", err)
			continue
		}

		err = n.state.RemoveBlocks(forkedBlock)

		if err != nil {
			fmt.Printf("ERROR: %s\n", err)
			continue
		}
	}
}

func (n *Node) getBlocksFromPeer(peer PeerNode) ([]database.Block, error) {
	url := fmt.Sprintf(
		"%s://%s%s",
		peer.ApiProtocol(),
		peer.TcpAddress(),
		endpointListBlocks,
	)

	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	blocks := make([]database.Block, 0)
	err = readResponse(res, &blocks)
	if err != nil {
		return nil, err
	}

	return blocks, nil
}
