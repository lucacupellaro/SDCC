package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"log"
	"os"
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

func removeAndSortMe(bucket [][]byte, selfId []byte) [][]byte {
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
