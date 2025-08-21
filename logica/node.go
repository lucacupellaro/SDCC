package logica

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	pb "kademlia-nft/proto/kad"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// struttura per il salvataggio
type KBucketFile struct {
	NodeID    string   `json:"node_id"`
	BucketHex []string `json:"bucket_hex"`
	SavedAt   string   `json:"saved_at"`
}

// SaveKBucket salva il bucket del nodo su file JSON
func SaveKBucket(nodeID string, bucket [][]byte, path string) error {
	// converte ogni []byte in stringa hex
	bucketHex := make([]string, len(bucket))
	for i, b := range bucket {
		bucketHex[i] = hex.EncodeToString(b)
	}

	data := KBucketFile{
		NodeID:    nodeID,
		BucketHex: bucketHex,
		SavedAt:   time.Now().UTC().Format(time.RFC3339),
	}

	// codifica in JSON con indentazione
	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	// scrive il file
	return os.WriteFile(path, jsonBytes, 0o644)
}

func RemoveAndSortMe(bucket [][]byte, selfId []byte) [][]byte {
	// Rimuove un nodo dal bucket
	for i := range bucket {
		if bytes.Equal(bucket[i], selfId) {
			bucket = append(bucket[:i], bucket[i+1:]...)
			break
		}
	}

	// 2. Riordina per distanza XOR dal nodo corrente (selfID)
	sort.Slice(bucket, func(i, j int) bool {
		distI := XOR(selfId, bucket[i])
		distJ := XOR(selfId, bucket[j])
		return LessThan(distI, distJ)
	})

	return bucket

}

type KademliaServer struct {
	pb.UnimplementedKademliaServer
}

func (s *KademliaServer) GetNodeList(ctx context.Context, req *pb.GetNodeListReq) (*pb.GetNodeListRes, error) {
	raw := os.Getenv("NODES")
	if raw == "" {
		log.Println("WARN: NODES env vuota nel seeder")
		return &pb.GetNodeListRes{}, nil
	}
	parts := strings.Split(raw, ",")
	out := &pb.GetNodeListRes{Nodes: make([]*pb.Node, 0, len(parts))}
	for _, name := range parts {
		out.Nodes = append(out.Nodes, &pb.Node{
			Id:   name,
			Host: name,
			Port: 8000,
		})
	}
	return out, nil
}

func GetNodeListIDs(seederAddr, requesterID string) ([]string, error) {
	// 1) connetti al seeder via gRPC
	conn, err := grpc.Dial(seederAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	// 2) crea client e manda la richiesta
	client := pb.NewKademliaClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	resp, err := client.GetNodeList(ctx, &pb.GetNodeListReq{RequesterId: requesterID})
	if err != nil {
		return nil, err
	}

	// 3) mappa la lista in []string con gli ID
	ids := make([]string, 0, len(resp.Nodes))
	for _, n := range resp.Nodes {
		ids = append(ids, n.Id)
	}
	return ids, nil
}

func resolveStartHostPort(name string) (string, error) {
	name = strings.TrimSpace(strings.ToLower(name))
	// supporta sia "node3" sia "nodo3"
	if strings.HasPrefix(name, "nodo") {
		name = "node" + name[len("nodo"):]
	}
	var n int
	if _, err := fmt.Sscanf(name, "node%d", &n); err != nil || n < 1 || n > 11 {
		return "", fmt.Errorf("nome nodo non valido: %q", name)
	}
	// La CLI corre su HOST → usa la porta mappata localhost:800N
	return fmt.Sprintf("localhost:%d", 8000+n), nil
}

func (s *KademliaServer) GetKBucket(ctx context.Context, req *pb.GetKBucketReq) (*pb.GetKBucketResp, error) {
	// 1) Path corretto nel container
	dataDir := strings.TrimSpace(os.Getenv("DATA_DIR"))
	if dataDir == "" {
		dataDir = "/data"
	}
	kbPath := filepath.Join(dataDir, "kbucket.json")

	// 2) Leggi file
	kbBytes, err := os.ReadFile(kbPath)
	if err != nil {
		return nil, fmt.Errorf("errore lettura %s: %w", kbPath, err)
	}

	// 3) Mappa JSON reale
	var kb struct {
		NodeID    string   `json:"node_id"`
		BucketHex []string `json:"bucket_hex"`
		SavedAt   string   `json:"saved_at"`
	}
	if err := json.Unmarshal(kbBytes, &kb); err != nil {
		return nil, fmt.Errorf("errore parse kbucket.json: %w", err)
	}

	// 4) Converte hex -> "nodeX"
	nodes := make([]*pb.Node, 0, len(kb.BucketHex))
	for _, hx := range kb.BucketHex {
		b, err := hex.DecodeString(hx)
		if err != nil || len(b) == 0 {
			continue
		}
		id := string(bytes.TrimRight(b, "\x00")) // es. "node4"
		if id == "" {
			continue
		}
		nodes = append(nodes, &pb.Node{
			Id:   id,
			Host: id,   // così la CLI può usare direttamente l’Id come host
			Port: 8000, // porta interna del servizio gRPC nei container
		})
	}

	return &pb.GetKBucketResp{Nodes: nodes}, nil
}

func (s *KademliaServer) Ping(ctx context.Context, req *pb.PingReq) (*pb.PingRes, error) {
	if f := req.GetFrom(); f != nil && f.GetId() != "" {
		log.Printf("[Ping] ricevuto From.Id=%q", f.GetId())
		if err := TouchContact(f.GetId()); err != nil {
			log.Printf("[Ping] TouchContact(%q) FAILED: %v", f.GetId(), err)
		} else {
			log.Printf("[Ping] TouchContact(%q) OK (bucket aggiornato)", f.GetId())
		}
	} else {
		log.Printf("[Ping] req.From mancante o vuoto: nessun update del bucket")
	}

	self := os.Getenv("NODE_ID")
	if self == "" {
		self = "unknown"
	}
	return &pb.PingRes{Ok: true, NodeId: self, UnixMs: time.Now().UnixMilli()}, nil
}

func (s *KademliaServer) UpdateBucket(ctx context.Context, req *pb.UpdateBucketReq) (*pb.UpdateBucketRes, error) {
	c := req.GetContact()
	if c == nil || c.GetId() == "" {
		return &pb.UpdateBucketRes{Ok: false}, nil
	}
	if err := TouchContact(c.GetId()); err != nil {
		return nil, err
	}
	return &pb.UpdateBucketRes{Ok: true}, nil
}

// ---------------------
// calcola l’hex da "nodeX" con la tua stessa regola dei 20 byte
func idHexFromNodeID(nodeID string) string {
	b := NewIDFromToken(nodeID, 20)
	return hex.EncodeToString(b)
}

const (
	kBucketPath = "/data/kbucket.json"
	kCapacity   = 8
)

type kbucketFile struct {
	NodeID    string   `json:"node_id"`
	BucketHex []string `json:"bucket_hex"`
	SavedAt   string   `json:"saved_at"`
}

func loadKBucket(path string) (kb kbucketFile, _ error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return kb, err
	}
	err = json.Unmarshal(b, &kb)
	return kb, err
}

func saveKBucket(path string, kb kbucketFile) error {
	kb.SavedAt = time.Now().UTC().Format(time.RFC3339)
	j, _ := json.MarshalIndent(kb, "", "  ")
	return os.WriteFile(path, j, 0o644)
}

func touchContactHex(kb *kbucketFile, hexID string) {
	// rimuovi se presente
	out := kb.BucketHex[:0]
	for _, h := range kb.BucketHex {
		if h != hexID {
			out = append(out, h)
		}
	}
	kb.BucketHex = out

	// append in coda, con capacità
	if len(kb.BucketHex) < kCapacity {
		kb.BucketHex = append(kb.BucketHex, hexID)
		return
	}
	// bucket pieno: drop LRU (pos 0) e append nuovo
	kb.BucketHex = append(kb.BucketHex[1:], hexID)
}

func TouchContact(nodeID string) error {
	hexID := idHexFromNodeID(nodeID)
	kb, err := loadKBucket(kBucketPath)
	if err != nil {
		return err
	}
	touchContactHex(&kb, hexID)
	return saveKBucket(kBucketPath, kb)
}
