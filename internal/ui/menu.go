package ui

import (
	"bufio"
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	pb "kademlia-nft/proto/kad"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type MenuChoice int

const (
	MenuListNodes MenuChoice = iota + 1
	MenuShowBucket
	MenuUseNode
	MenuSearchNFT
	MenuShowEdges
	MenuQuit
)

type KademliaServer struct {
	pb.UnimplementedKademliaServer
}

func ShowWelcomeMenu() MenuChoice {
	fmt.Print("\033[2J\033[H") // clear screen
	fmt.Println(`
╔══════════════════════════════════════════════╗
║        Kademlia NFT – Console Control        ║
╚══════════════════════════════════════════════╝
Benvenuto! Seleziona un'operazione:

  1) Elenca nodi
  2) Mostra k-bucket di un nodo
  3) Seleziona nodo 
  4) Cerca NFT a partire dal nodo corrente
  5) Mostra collegamenti (edges)
  6) Esci
`)

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("Scegli [1-6]: ")
		line, _ := reader.ReadString('\n')
		line = strings.TrimSpace(line)
		switch line {
		case "1":
			return MenuListNodes
		case "2":
			return MenuShowBucket
		case "3":
			return MenuUseNode
		case "4":
			return MenuSearchNFT
		case "5":
			return MenuShowEdges
		case "6", "q", "Q", "exit", "quit":
			return MenuQuit
		default:
			fmt.Println("Scelta non valida, riprova.")
		}
	}
}

// Restituisce i servizi Compose attivi (node1, node2, ...) del progetto.
func ListActiveComposeServices(project string) ([]string, error) {
	// Filtra per progetto per non mischiare altri container.
	// --format {{.Names}} per ottenere i nomi dei container es.: kademlia-nft-node1-1
	cmd := exec.Command("docker", "ps",
		"--filter", "label=com.docker.compose.project="+project,
		"--format", "{{.Names}}",
	)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	services := make([]string, 0, len(lines))
	for _, name := range lines {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		// Nome Compose tipico: <project>-<service>-<index>
		parts := strings.Split(name, "-")
		if len(parts) >= 3 {
			services = append(services, parts[len(parts)-2]) // prende <service> (es. "node1")
		}
	}
	return dedupe(services), nil
}

func dedupe(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

func LookupNFTClient(startNodeHostPort string, key []byte, maxHops int) (*pb.LookupNFTRes, error) {
	visited := make(map[string]bool)
	toVisit := []string{startNodeHostPort}

	for hops := 0; hops < maxHops && len(toVisit) > 0; hops++ {
		current := toVisit[0]
		toVisit = toVisit[1:]

		if visited[current] {
			continue
		}
		visited[current] = true

		// gRPC call
		conn, err := grpc.Dial(current, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			continue // prova gli altri
		}
		client := pb.NewKademliaClient(conn)
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		resp, err := client.LookupNFT(ctx, &pb.LookupNFTReq{
			FromId: os.Getenv("NODE_ID"),
			Key:    &pb.Key{Key: key},
		})
		cancel()
		_ = conn.Close()

		if err != nil {
			continue // prova gli altri
		}

		if resp.GetFound() {
			return resp, nil // trovato!
		}

		// Aggiungo i suggeriti (nearest) in coda
		for _, n := range resp.GetNearest() {
			addr := fmt.Sprintf("%s:%d", n.Host, n.Port) // es. "node7:8000"
			if !visited[addr] {
				toVisit = append(toVisit, addr)
			}
		}
	}

	return nil, fmt.Errorf("NFT non trovato (hops=%d)", maxHops)
}

// dentro lo stesso file dove hai type KademliaServer {...}

func (s *KademliaServer) LookupNFT(ctx context.Context, req *pb.LookupNFTReq) (*pb.LookupNFTRes, error) {
	// 1) cerco localmente
	dataDir := strings.TrimSpace(os.Getenv("DATA_DIR"))
	if dataDir == "" {
		dataDir = "/data"
	}

	fileName := fmt.Sprintf("%x.json", req.Key.Key)
	fmt.Printf("stiamo cercando il file%s\n", fileName)
	filePath := filepath.Join(dataDir, fileName)

	if b, err := os.ReadFile(filePath); err == nil {
		// trovato localmente
		return &pb.LookupNFTRes{
			Found:  true,
			Holder: &pb.Node{Id: os.Getenv("NODE_ID"), Host: os.Getenv("NODE_ID"), Port: 8000},
			Value:  &pb.NFTValue{Bytes: b},
			// nearest vuoto se trovato
		}, nil
	}

	// 2) non trovato: restituisco i vicini dal kbucket, ordinati per distanza dalla chiave
	//    - leggo /data/kbucket.json
	type KBucketFile struct {
		NodeID    string   `json:"node_id"`
		BucketHex []string `json:"bucket_hex"`
		SavedAt   string   `json:"saved_at"`
	}
	kbPath := filepath.Join(dataDir, "kbucket.json")
	kbBytes, err := os.ReadFile(kbPath)
	if err != nil {
		// non ho kbucket -> non posso suggerire vicini
		return &pb.LookupNFTRes{Found: false}, nil
	}
	var kb KBucketFile
	if err := json.Unmarshal(kbBytes, &kb); err != nil {
		return &pb.LookupNFTRes{Found: false}, nil
	}

	// decodifico gli ID in []byte
	var bucket [][]byte
	for _, hx := range kb.BucketHex {
		b, decErr := hex.DecodeString(hx)
		if decErr == nil {
			bucket = append(bucket, b)
		}
	}

	// ordino i bucket per distanza XOR dalla chiave
	// NB: usa le tue funzioni XOR/LessThan già presenti
	type pair struct {
		id   []byte
		dist []byte
	}
	pairs := make([]pair, 0, len(bucket))
	for _, nid := range bucket {
		pairs = append(pairs, pair{id: nid, dist: XOR(req.Key.Key, nid)})
	}
	sort.Slice(pairs, func(i, j int) bool { return LessThan(pairs[i].dist, pairs[j].dist) })

	// prendo i primi N (es. 3–7) e li trasformo in pb.Node
	N := 5
	if N > len(pairs) {
		N = len(pairs)
	}
	nearest := make([]*pb.Node, 0, N)
	for i := 0; i < N; i++ {
		host := DecodeID(pairs[i].id) // es. "node7"
		nearest = append(nearest, &pb.Node{Id: host, Host: host, Port: 8000})
	}

	return &pb.LookupNFTRes{
		Found:   false,
		Nearest: nearest,
	}, nil
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
