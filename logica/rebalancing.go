package logica

import (
	"context"
	"encoding/json"
	pb "kademlia-nft/proto/kad"
	"path/filepath"

	"fmt"

	"encoding/hex"
	"os"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type TempNFT struct {
	TokenID         string `json:"token_id"`
	Name            string `json:"name"`
	Index           string `json:"index"`
	Volume          string `json:"volume"`
	VolumeUSD       string `json:"volume_usd"`
	MarketCap       string `json:"market_cap"`
	MarketCapUSD    string `json:"market_cap_usd"`
	Sales           string `json:"sales"`
	FloorPrice      string `json:"floor_price"`
	FloorPriceUSD   string `json:"floor_price_usd"`
	AveragePrice    string `json:"average_price"`
	AveragePriceUSD string `json:"average_price_usd"`
	Owners          string `json:"owners"`
	Assets          string `json:"assets"`
	OwnerAssetRatio string `json:"owner_asset_ratio"`
	Category        string `json:"category"`
	Website         string `json:"website"`
	Logo            string `json:"logo"`
}

// helper: normalizza host/porta (porta 0 o vuota -> 8000)
func sanitizeHostPort(host string, port int) (string, int) {
	host = strings.TrimSpace(host)
	if host == "" {
		host = "localhost"
	}
	if port == 0 {
		port = 8000
	}
	return host, port
}
func (s *KademliaServer) Rebalance(ctx context.Context, req *pb.RebalanceReq) (*pb.RebalanceRes, error) {
	nodo := strings.TrimSpace(req.GetTargetId())
	k := int(req.GetK())
	if k <= 0 {
		k = 2
	}

	// --- Rubrica chiave-stabile -> endpoint (host:port) + lista chiavi per mapping ---
	type hostPort struct {
		Host string
		Port int
	}
	peerAddr := make(map[string]hostPort, len(req.GetNodes()))
	nodeKeys := make([]string, 0, len(req.GetNodes())) // SOLO chiavi stabili per il mapping (no :port)

	for _, n := range req.GetNodes() {
		key := strings.TrimSpace(n.GetId())
		if key == "" {
			key = strings.TrimSpace(n.GetHost())
		}
		if key == "" {
			continue
		}
		h, p := sanitizeHostPort(n.GetHost(), int(n.GetPort()))
		peerAddr[key] = hostPort{Host: h, Port: p}
		nodeKeys = append(nodeKeys, key)
	}

	// --- Byte mapping su chiavi stabili ---
	dir := BuildByteMappingSHA1(nodeKeys)
	fmt.Printf("ByteMapping costruito su chiavi: %v\n", nodeKeys)

	// --- helper: controlla presenza NFT su un nodo via LookupNFT ---
	hasNFT := func(addr string, tokenID []byte) (bool, error) {
		cctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		conn, err := grpc.DialContext(cctx, addr, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
		if err != nil {
			return false, fmt.Errorf("dial %s: %w", addr, err)
		}
		defer conn.Close()

		client := pb.NewKademliaClient(conn)
		resp, err := client.LookupNFT(cctx, &pb.LookupNFTReq{
			FromId: nodo,
			Key:    &pb.Key{Key: tokenID},
		})
		if err != nil {
			return false, err
		}
		return resp.GetFound(), nil
	}

	// --- scan directory dati ---
	dataDir := strings.TrimSpace(os.Getenv("DATA_DIR"))
	if dataDir == "" {
		dataDir = "/data"
	}
	entries, err := os.ReadDir(dataDir)
	if err != nil {
		return nil, fmt.Errorf("ReadDir(%s): %w", dataDir, err)
	}
	fmt.Printf("[Rebalance] dirPath=%s entries=%d\n", dataDir, len(entries))

	var moved, kept int
	var skippedNonJSON, skippedReadErr, skippedParseErr, skippedBadToken, skippedNoAssigned int

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.ToLower(filepath.Ext(e.Name())) != ".json" {
			skippedNonJSON++
			continue
		}

		path := filepath.Join(dataDir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			skippedReadErr++
			fmt.Printf("⚠️ ReadFile(%s): %v\n", path, err)
			continue
		}

		var tmp TempNFT
		if err := json.Unmarshal(data, &tmp); err != nil {
			skippedParseErr++
			fmt.Printf("⚠️ Unmarshal(%s): %v\n", e.Name(), err)
			continue
		}

		// Token stabile: SHA1 del Nome (coerente col resto del codice)
		tokenID := Sha1ID(tmp.Name)

		// Nodi assegnati (k più vicini)
		assigned := ClosestNodesForNFTWithDir(tokenID, dir, k)
		if len(assigned) == 0 {
			skippedNoAssigned++
			fmt.Printf("⚠️ %s: nessun nodo assegnato per token %q → skip\n", e.Name(), tmp.Name)
			continue
		}

		// Endpoint reali (host:port) per i nodi assegnati
		type dest struct{ name, addr string } // name = chiave stabile o hostname del nodo
		dests := make([]dest, 0, len(assigned))
		for _, a := range assigned {
			if hp, ok := peerAddr[a.Key]; ok {
				dests = append(dests, dest{name: a.Key, addr: fmt.Sprintf("%s:%d", hp.Host, hp.Port)})
			} else {
				// fallback sicuro
				dests = append(dests, dest{name: a.Key, addr: fmt.Sprintf("%s:%d", a.Key, 8000)})
			}
		}

		// Log dei due più vicini (nomi/chiavi, non esadecimali grezzi)
		names := make([]string, 0, len(dests))
		for _, d := range dests {
			names = append(names, d.name)
		}
		fmt.Printf("assegnati per %q → %v\n", tmp.Name, names)

		// Verifica presenza su ciascun nodo assegnato
		present := make([]bool, len(dests))
		for i, d := range dests {
			ok, err := hasNFT(d.addr, tokenID)
			if err != nil {
				fmt.Printf("ℹ️ Lookup su %s fallito: %v\n", d.addr, err)
				present[i] = false
				continue
			}
			present[i] = ok
		}

		// Chi manca?
		missingAddrs := make([]string, 0, len(dests))
		for i, ok := range present {
			if !ok {
				missingAddrs = append(missingAddrs, dests[i].addr)
			}
		}

		// Il nodo corrente è tra i 2 assegnati?
		nodeIsAssigned := false
		for _, d := range dests {
			// confrontiamo con l'identità che usi come TargetId (es. "node6")
			if d.name == nodo || d.name == peerAddr[nodo].Host {
				nodeIsAssigned = true
				break
			}
		}

		switch {
		case len(missingAddrs) == 0:
			// già presente su TUTTI i nodi assegnati
			if !nodeIsAssigned {
				// il nodo corrente non è tra i più vicini → elimina la copia locale
				if err := os.Remove(path); err != nil {
					fmt.Printf("⚠️ Remove(%s): %v\n", path, err)
					continue
				}
				moved++
			} else {
				kept++
			}

		default:
			// mancano repliche: replichiamo SOLO sui mancanti
			finale := convert(NFT{}, tmp, nil)
			if err := StoreNFTToNodes(finale, tokenID, finale.Name, missingAddrs, 24*3600); err != nil {
				fmt.Printf("❌ Replicazione NFT %q fallita (dest=%v): %v\n", tmp.Name, missingAddrs, err)
				// non rimuovere la copia locale in caso di errore
				continue
			}
			// dopo replica, se questo nodo NON è tra gli assegnati, elimina locale
			if !nodeIsAssigned {
				if err := os.Remove(path); err != nil {
					fmt.Printf("⚠️ Remove(%s): %v\n", path, err)
					continue
				}
				moved++
			} else {
				kept++
			}
		}
	}

	msg := fmt.Sprintf(
		"Nodo %s: %d NFT tenuti, %d spostati. skipped: nonjson=%d read=%d parse=%d badtoken=%d noassigned=%d",
		nodo, kept, moved, skippedNonJSON, skippedReadErr, skippedParseErr, skippedBadToken, skippedNoAssigned,
	)
	return &pb.RebalanceRes{
		Moved:   int32(moved),
		Kept:    int32(kept),
		Message: msg,
	}, nil
}

func convert(to NFT, from TempNFT, nodiSelected []string) NFT {

	fmt.Printf("ID NFT: %s\n", from.TokenID)
	tokenID, _ := hex.DecodeString(from.TokenID)

	to.TokenID = tokenID
	to.Name = from.Name
	to.Index = from.Index
	to.Volume = from.Volume
	to.Volume_USD = from.VolumeUSD
	to.Market_Cap = from.MarketCap
	to.Market_Cap_USD = from.MarketCapUSD
	to.Sales = from.Sales
	to.Floor_Price = from.FloorPrice
	to.Floor_Price_USD = from.FloorPriceUSD
	to.Average_Price = from.AveragePrice
	to.Average_Price_USD = from.AveragePriceUSD
	to.Owners = from.Owners
	to.Assets = from.Assets
	to.Owner_Asset_Ratio = from.OwnerAssetRatio
	to.Category = from.Category
	to.Website = from.Website
	to.Logo = from.Logo
	to.AssignedNodesToken = nodiSelected

	return to
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
		targetAddr, // es: "localhost:8006"
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
		host, portStr, _ := strings.Cut(n, ":") // es: "node6:8000" → host="node6"
		port, _ := strconv.Atoi(portStr)

		pbNodes = append(pbNodes, &pb.Node{
			Id:   host, // <-- USA l’ID “umano” (nodeX)
			Host: host,
			Port: int32(port),
		})
	}

	//fmt.Printf("%+v\n", pbNodes)
	//fmt.Printf("%+v\n", targetID)

	req := &pb.RebalanceReq{
		TargetId: targetID, //  nodo target
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
