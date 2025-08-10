package main

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"sort"
)

type ID [20]byte //160 bit ID

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
	for i := 1; i < len(records); i++ { // <-- parte da 1
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
