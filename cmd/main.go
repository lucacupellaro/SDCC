package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

type NFT struct {
	// colonne CSV
	Index             string
	Name              string
	Volume            string
	Volume_USD        string
	Market_Cap        string
	Market_Cap_USD    string
	Sales             string
	Floor_Price       string
	Floor_Price_USD   string
	Average_Price     string
	Average_Price_USD string
	Owners            string
	Assets            string
	Owner_Asset_Ratio string
	Category          string
	Website           string
	Logo              string

	// campi interni al tuo sistema
	TokenID            []byte
	AssignedNodesToken [][]byte
}

func main() {

	var csvAll [][]string

	go func() {
		if err := runGRPCServer(); err != nil {
			log.Printf("gRPC server chiuso: %v", err)
		}
	}()

	// opzionale: piccolo delay per dare tempo al listener di alzarsi
	time.Sleep(400 * time.Millisecond)

	// Prende l'ID del nodo dall'ambiente
	nodeID := os.Getenv("NODE_ID")
	if nodeID == "" {
		nodeID = "default"
	}

	fmt.Println("Avviato nodo:", nodeID)

	isSeeder := os.Getenv("SEED") == "true"

	if isSeeder {

		fmt.Printf("sono il seeder,\nReading CSV file...\n")

		csvAll = readCsv2("csv/NFT_Top_Collections.csv")

		fmt.Printf("NFT letti: %d\n", len(csvAll))

		// Genera gli ID per la lista di NFT
		fmt.Printf("Generating IDs for NFTs...\n")

		var colName []string
		for i, row := range csvAll {
			if i == 0 {
				continue // salta intestazione se presente
			}
			colName = append(colName, row[1])
		}

		fmt.Printf("NFT 0z %s\n", colName[0])

		listNFTId := generateBytesOfAllNfts(colName)

		fmt.Printf("NFT id %x: NFT name: %s\n", listNFTId[0], DecodeID(listNFTId[0]))

		// recuper gli ID dei container
		rawNodes := os.Getenv("NODES")
		if rawNodes == "" {
			fmt.Println("Nessun nodo in NODES")
			return
		}

		parts := strings.Split(rawNodes, ",")
		iDnew := make([][]byte, len(parts))
		for i, p := range parts {
			//fmt.Println("Nodo trovato:", p)
			iDnew[i] = NewIDFromToken(p, 20)

		}

		//fissato nft quindi per ogni nft , crei unafunzione che per ogni nft scorre tutti e e 10 gli i d dei nodi e li assegna ai 2/3 pi vicini)

		fmt.Println("Assegnazione dei k nodeID più vicini agli NFT...")

		rows := csvAll[1:] // salta header
		nfts := make([]NFT, 0, len(rows))
		for _, row := range rows {
			if len(row) < 17 {
				continue
			} // safety

			name := strings.TrimSpace(row[1])
			key := NewIDFromToken(name, 20) // ID dal Name (come vuoi tu)

			col := func(k int) string {
				if k >= 0 && k < len(row) {
					return strings.TrimSpace(row[k])
				}
				return ""
			}

			nfts = append(nfts, NFT{
				Index:             col(0),
				Name:              name,
				Volume:            col(2),
				Volume_USD:        col(3),
				Market_Cap:        col(4),
				Market_Cap_USD:    col(5),
				Sales:             col(6),
				Floor_Price:       col(7),
				Floor_Price_USD:   col(8),
				Average_Price:     col(9),
				Average_Price_USD: col(10),
				Owners:            col(11),
				Assets:            col(12),
				Owner_Asset_Ratio: col(13),
				Category:          col(14),
				Website:           col(15),
				Logo:              col(16),

				TokenID:            key,
				AssignedNodesToken: AssignNFTToNodes(key, iDnew, 2),
			})
		}

		for _, h := range parts {
			if err := waitReady(h, 12*time.Second); err != nil {
				log.Fatalf("❌ Nodo %s non pronto: %v", h, err) // fermati se uno non è pronto
			}
		}

		//-------------Salvatggio degli NFT sugli appositi Nodi-------------------------------------------------------------------//

		fmt.Printf("struct size: %d\n", len(nfts))

		for j := 0; j < len(nfts); j++ {
			var nodi []string
			nodi = append(nodi, DecodeID(nfts[j].AssignedNodesToken[0]))
			nodi = append(nodi, DecodeID(nfts[j].AssignedNodesToken[1]))

			if err := StoreNFTToNodes(nfts[j], DecodeID(nfts[j].TokenID), nfts[j].Name, nodi, 24*3600); err != nil {
				fmt.Println("Errore:", err)
				continue
			}

			//fmt.Printf("Salvati NFT numero: %d\n", j)

			nodi = nil

		}

		select {} // blocca per sempre

	} else {

		var nodes []string
		var TokenNodo []byte
		var Bucket [][]byte
		var BucketSort [][]byte

		//-------------------I container si mettono in ascolto qui-------------------//

		nodeID := os.Getenv("NODE_ID")
		if nodeID == "" {
			nodeID = "default"
		}

		TokenNodo = NewIDFromToken(nodeID, 20)

		fmt.Printf("Sono il nodo %s, PID: %d\n", DecodeID(TokenNodo), os.Getpid())

		//---------Recuperlo la lista dei nodi chiedendola al Seeder-------------------------
		nodes, err := GetNodeListIDs("node1:8000", os.Getenv("NODE_ID"))

		if err != nil {
			log.Fatalf("Errore recupero nodi dal seeder: %v", err)
		}

		var nodiTokenizati [][]byte
		for i := 0; i < len(nodes); i++ {
			nodiTokenizati = append(nodiTokenizati, NewIDFromToken(nodes[i], 20))
		}

		//--------------------Ogni container si trova i k bucket piu vicini e li salva nel proprio volume-------------------//

		Bucket = AssignNFTToNodes(TokenNodo, nodiTokenizati, 8)

		BucketSort = removeAndSortMe(Bucket, TokenNodo)

		fmt.Printf("sto salvando il kbucket per il nodo,%s\n", DecodeID(TokenNodo))

		err2 := SaveKBucket(nodeID, BucketSort, "/data/kbucket.json")

		if err2 != nil {
			log.Fatalf("Errore salvataggio K-bucket: %v", err)
		}

		select {} // blocca per sempre

	}

}
