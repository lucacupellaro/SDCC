package logica

import (
	"context"
	"encoding/json"
	pb "kademlia-nft/proto/kad"
	"path/filepath"

	"fmt"

	"os"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func (s *KademliaServer) Rebalance(ctx context.Context, req *pb.RebalanceReq) (*pb.RebalanceRes, error) {
	nodo := req.TargetId
	k := int(req.K)
	if k <= 0 {
		k = 2
	}

	// 🔧 ricostruzione ByteMapping dai nodi ricevuti
	var nodeStrings []string
	for _, n := range req.Nodes {
		nodeStrings = append(nodeStrings, fmt.Sprintf("%s:%d", n.Host, n.Port))
	}
	dir := BuildByteMappingSHA1(nodeStrings)

	// 📂 scansiono i file NFT locali
	dataDir := strings.TrimSpace(os.Getenv("DATA_DIR"))
	if dataDir == "" {
		dataDir = "/data"
	}
	dirPath := dataDir // ✅ /data è già la root del nodo (./data/node6 montata su /data)

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, fmt.Errorf("ReadDir(%s): %w", dirPath, err)
	}
	fmt.Printf("[Rebalance] dirPath=%s entries=%d\n", dirPath, len(entries))

	var moved, kept int
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		path := filepath.Join(dirPath, e.Name())
		data, _ := os.ReadFile(path)

		var nft NFT
		if err := json.Unmarshal(data, &nft); err != nil {
			continue
		}

		assigned := ClosestNodesForNFTWithDir(nft.TokenID, dir, k)

		if NFTBelongsHere(nodo, assigned) {
			kept++
		} else {
			moved++
			var nodi []string
			for _, a := range assigned {
				nodi = append(nodi, a.Key) // Key = "nodeX"
			}
			// 📤 salvo NFT nei nodi corretti
			err := StoreNFTToNodes(nft, nft.TokenID, nft.Name, nodi, 24*3600)
			if err != nil {
				fmt.Printf("Errore spostamento NFT %s: %v\n", nft.TokenID, err)
				continue
			}
			_ = os.Remove(path) // rimuovo dal nodo corrente
		}
	}

	return &pb.RebalanceRes{
		Moved:   int32(moved),
		Kept:    int32(kept),
		Message: fmt.Sprintf("Nodo %s: %d NFT tenuti, %d spostati", nodo, kept, moved),
	}, nil
}

func NFTBelongsHere(nodo string, assigned []NodePick) bool {
	for _, a := range assigned {
		if a.Key == nodo { // Key contiene il nome del nodo, es: "node4"
			return true
		}
	}
	return false
}

func RebalanceNode(targetAddr string, targetID string, activeNodes []string, k int) error {

	dctx, dcancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer dcancel()

	conn, err := grpc.DialContext(
		dctx,
		targetAddr, // es. "node6:8000" se dashboard è in rete docker; altrimenti "localhost:8006"
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return fmt.Errorf("DIAL FALLITA verso %s: %w", targetAddr, err)
	}
	defer conn.Close()

	client := pb.NewKademliaClient(conn)

	var pbNodes []*pb.Node
	for _, n := range activeNodes {
		nodeID := Sha1ID(n)                     // ID Kademlia (20 byte → hex o base64)
		host, portStr, _ := strings.Cut(n, ":") // separa host:port
		port, _ := strconv.Atoi(portStr)

		pbNodes = append(pbNodes, &pb.Node{
			Id:   fmt.Sprintf("%x", nodeID),
			Host: host,
			Port: int32(port),
		})
	}

	fmt.Printf("%+v\n", pbNodes)
	fmt.Printf("%+v\n", targetID)

	req := &pb.RebalanceReq{
		TargetId: targetID, // ID Kademlia (SHA1) del nodo target
		Nodes:    pbNodes,  // lista di nodi attivi
		K:        int32(k), // numero repliche
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := client.Rebalance(ctx, req)
	if err != nil {
		return fmt.Errorf("errore chiamata Rebalance: %v", err)
	}

	fmt.Printf("✅ Rebalance completato per %s\n", targetID)
	fmt.Printf("   - NFT tenuti: %d\n", resp.Kept)
	fmt.Printf("   - NFT spostati: %d\n", resp.Moved)
	fmt.Println("   - Messaggio:", resp.Message)

	return nil
}
