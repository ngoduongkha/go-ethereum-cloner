package node

// import (
// 	"context"
// 	"fmt"
// 	"net/http"
// 	"time"

// 	"github.com/ngoduongkha/go-ethereum-cloner/database"
// )

// // Check state is forked and remove forked blocks
// func (n *Node) checkForkedState(ctx context.Context) error {
// 	n.doCheckForkedState()

// 	ticker := time.NewTicker(checkForkedStateIntervalSeconds * time.Second)

// 	for {
// 		select {
// 		case <-ticker.C:
// 			n.doCheckForkedState()

// 		case <-ctx.Done():
// 			ticker.Stop()
// 		}
// 	}
// }

// func (n *Node) doCheckForkedState() {
// 	if n.state.IsForked() {
// 		fmt.Println("State is forked. Removing forked blocks...")

// 		n.state.RemoveForkedBlocks()
// 	}
// }
