package logica

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	pb "kademlia-nft/proto/kad"

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
	self := os.Getenv("NODE_ID")
	if self == "" {
		self = "unknown"
	}

	// (opzionale ma consigliato per Kademlia) aggiorna la tua routing table con req.From
	// UpdateBucket(req.From.Id)  // TODO: la tua funzione di update+persist su kbucket.json

	return &pb.PingRes{
		Ok:     true,
		NodeId: self,
		UnixMs: time.Now().UnixMilli(),
	}, nil
}
