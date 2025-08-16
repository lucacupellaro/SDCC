package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "kademlia-nft/proto/kad"
)

func readCsv(path string) []string {

	//fmt.Printf("file: %s\n", path)

	file, err := os.Open(path)
	if err != nil {
		fmt.Printf("Errore nell'aprire il file CSV: %s\n", err)
		log.Fatal(err)
	}

	defer file.Close()

	reader := csv.NewReader(file)

	records, err := reader.ReadAll()
	if err != nil {
		fmt.Printf("Errore nella lettura del file CSV: %s\n", err)
		log.Fatal(err)
	}

	col1 := make([]string, 0, len(records))
	for i := 0; i < len(records); i++ { // <-- parte da 0
		row := records[i]
		if len(row) < 2 {
			fmt.Printf("Riga %d ha meno di 2 colonne\n", i)
			continue
		}
		col1 = append(col1, row[1])
	}

	fmt.Println("CSV file read successfully")
	return col1
}

func readCsv2(path string) [][]string {

	//fmt.Printf("file: %s\n", path)

	file, err := os.Open(path)
	if err != nil {
		fmt.Printf("Errore nell'aprire il file CSV: %s\n", err)
		log.Fatal(err)
	}

	defer file.Close()

	reader := csv.NewReader(file)

	records, err := reader.ReadAll()
	if err != nil {
		fmt.Printf("Errore nella lettura del file CSV: %s\n", err)
		log.Fatal(err)
	}

	fmt.Println("CSV file read successfully")
	return records
}

func NewIDFromToken(tokenID string, size int) []byte {
	b := []byte(tokenID)
	if len(b) > size {
		out := make([]byte, size)
		copy(out, b[:size])
		return out
	}
	padded := make([]byte, size)
	copy(padded, b)
	return padded
}

func DecodeID(b []byte) string {
	return string(bytes.TrimRight(b, "\x00"))
}

func XOR(a, b []byte) []byte {
	// assume stesse lunghezze; se no, usa la min
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	out := make([]byte, n)
	for i := 0; i < n; i++ {
		out[i] = a[i] ^ b[i]
	}
	return out
}

// confronto lessicografico: true se a < b
func LessThan(a, b []byte) bool {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		if a[i] < b[i] {
			return true
		}
		if a[i] > b[i] {
			return false
		}
	}
	// se prefissi uguali, quello più corto è “minore”
	return len(a) < len(b)
}

// Generate a list of IDs from a list of tokens or Nodes
func generateBytesOfAllNfts(list []string) [][]byte {
	ids := make([][]byte, len(list))
	for i, s := range list {
		ids[i] = NewIDFromToken(s, 20) // 20 bytes = 160 bit
	}
	return ids
}

// restituisce i k nodeID più vicini alla chiave (distanza XOR, ordinata crescente)
func AssignNFTToNodes(key []byte, nodes [][]byte, k int) [][]byte {
	if k <= 0 || len(nodes) == 0 {
		return nil
	}
	if k > len(nodes) {
		k = len(nodes)
	}

	type pair struct {
		id   []byte
		dist []byte
	}
	pairs := make([]pair, len(nodes))
	for i, nid := range nodes {
		pairs[i] = pair{id: nid, dist: XOR(key, nid)}
	}

	sort.Slice(pairs, func(i, j int) bool {
		return LessThan(pairs[i].dist, pairs[j].dist)
	})

	out := make([][]byte, k)
	for i := 0; i < k; i++ {
		out[i] = pairs[i].id
	}
	return out
}

// StoreNFTToNodes invia lo stesso NFT a tutti i nodi indicati.
// Ritorna nil se TUTTE le store vanno a buon fine; altrimenti un error descrittivo.
func StoreNFTToNodes(nft NFT, tokenID string, name string, nodes []string, ttlSecs int32) error {
	// chiave Kademlia a 20 byte (coerente con il resto del tuo codice)
	key := NewIDFromToken(tokenID, 20)

	// payload (serializzazione minimale dell'NFT)
	payload, _ := json.Marshal(struct {
		TokenID string `json:"token_id"`
		Name    string `json:"name"`

		Index string `json:"index,omitempty"`

		Volume            string `json:"volume,omitempty"`
		Volume_USD        string `json:"volume_usd,omitempty"`
		Market_Cap        string `json:"market_cap,omitempty"`
		Market_Cap_USD    string `json:"market_cap_usd,omitempty"`
		Sales             string `json:"sales,omitempty"`
		Floor_Price       string `json:"floor_price,omitempty"`
		Floor_Price_USD   string `json:"floor_price_usd,omitempty"`
		Average_Price     string `json:"average_price,omitempty"`
		Average_Price_USD string `json:"average_price_usd,omitempty"`
		Owners            string `json:"owners,omitempty"`
		Assets            string `json:"assets,omitempty"`
		Owner_Asset_Ratio string `json:"owner_asset_ratio,omitempty"`
		Category          string `json:"category,omitempty"`
		Website           string `json:"website,omitempty"`
		Logo              string `json:"logo,omitempty"`
	}{
		TokenID:           hex.EncodeToString(key),
		Name:              name,
		Index:             nft.Index,
		Volume:            nft.Volume,
		Volume_USD:        nft.Volume_USD,
		Market_Cap:        nft.Market_Cap,
		Market_Cap_USD:    nft.Market_Cap_USD,
		Sales:             nft.Sales,
		Floor_Price:       nft.Floor_Price,
		Floor_Price_USD:   nft.Floor_Price_USD,
		Average_Price:     nft.Average_Price,
		Average_Price_USD: nft.Average_Price_USD,
		Owners:            nft.Owners,
		Assets:            nft.Assets,
		Owner_Asset_Ratio: nft.Owner_Asset_Ratio,
		Category:          nft.Category,
		Website:           nft.Website,
		Logo:              nft.Logo,
	})

	var errs []string

	for _, host := range nodes {
		if strings.TrimSpace(host) == "" {
			errs = append(errs, "host vuoto")
			continue
		}

		addr := fmt.Sprintf("%s:%d", host, 8000)

		conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			errs = append(errs, fmt.Sprintf("dial %s: %v", addr, err))
			continue
		}

		client := pb.NewKademliaClient(conn)
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		_, callErr := client.Store(ctx, &pb.StoreReq{
			From:    &pb.Node{Id: "seeder", Host: "seeder", Port: 8000},
			Key:     &pb.Key{Key: key},
			Value:   &pb.NFTValue{Bytes: payload},
			TtlSecs: ttlSecs, // es: 24*3600. Metti 0 se non usi TTL.
		})
		cancel()
		_ = conn.Close()

		if callErr != nil {
			errs = append(errs, fmt.Sprintf("Store(%s): %v", host, callErr))
			continue
		}

		fmt.Printf("✅  NFT inviato %q su %s\n", tokenID, host)
	}

	if len(errs) > 0 {
		return fmt.Errorf("alcune Store sono fallite: %s", strings.Join(errs, "; "))
	}
	return nil // tutto ok
}

// ===== Server RPC =====

// Store implementa il metodo Store del servizio Kademlia.
func (s *KademliaServer) Store(ctx context.Context, req *pb.StoreReq) (*pb.StoreRes, error) {
	dataDir := strings.TrimSpace(os.Getenv("DATA_DIR"))
	if dataDir == "" {
		dataDir = "/data" // default nel container
	}
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("creazione dir %s: %w", dataDir, err)
	}

	fileName := fmt.Sprintf("%x.json", req.Key.Key)
	filePath := filepath.Join(dataDir, fileName)

	// (facoltativo) log utile per conferma
	abs, _ := filepath.Abs(filePath)
	log.Printf("✅ Salvato NFT in %s (abs=%s)", filePath, abs)

	if err := os.WriteFile(filePath, req.Value.Bytes, 0644); err != nil {
		return nil, fmt.Errorf("scrittura file %s: %w", filePath, err)
	}
	return &pb.StoreRes{Ok: true}, nil
}

func runGRPCServer() error {
	lis, err := net.Listen("tcp", ":8000")
	if err != nil {
		return err
	}
	gs := grpc.NewServer()
	pb.RegisterKademliaServer(gs, &KademliaServer{})
	log.Println("gRPC server in ascolto su :8000")
	return gs.Serve(lis) // BLOCCA
}

func waitReady(host string, timeout time.Duration) error {
	addr := fmt.Sprintf("%s:8000", host)
	deadline := time.Now().Add(timeout)
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		_, err := grpc.DialContext(
			ctx, addr,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithBlock(),
			grpc.WithReturnConnectionError(),
		)
		cancel()
		if err == nil {
			return nil // è raggiungibile
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timeout aspettando %s", addr)
		}
		time.Sleep(300 * time.Millisecond)
	}
}

// Su HOST: elenca i file in ./data/<nodeID>
// (se la esegui DENTRO al container, usa base := "/data" e ignora nodeID)
func ListNodeVolumeFiles(nodeID string) ([]string, error) {
	base := "./data" // nel container metti "/data"
	dir := filepath.Join(base, nodeID)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)
	return files, nil
}
