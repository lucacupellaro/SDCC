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

	var listNFT []string
	var listNFTId [][]byte

	var csvAll [][]string

	// Prende l'ID del nodo dall'ambiente
	nodeID := os.Getenv("NODE_ID")
	if nodeID == "" {
		nodeID = "default"
	}

	fmt.Println("Avviato nodo:", nodeID, "PID:", os.Getpid())

	isSeeder := os.Getenv("SEED") == "true"

	if isSeeder {

		go func() {
			if err := runGRPCServer(); err != nil {
				log.Fatalf("gRPC server error (seeder): %v", err)
			}
		}()

		time.Sleep(300 * time.Millisecond)

		fmt.Printf("sono il seeder,\nReading CSV file...\n")

		listNFT = readCsv("csv/NFT_Top_Collections.csv")
		csvAll = readCsv2("csv/NFT_Top_Collections.csv")

		fmt.Printf("NFT letti: %d\n", len(listNFT))

		// Genera gli ID per la lista di NFT
		fmt.Printf("Generating IDs for NFTs...\n")

		listNFTId = generateBytesOfAllNfts(listNFT)

		fmt.Printf("NFT id %x: NFT String: %s\n", listNFTId[0], DecodeID(listNFTId[0]))

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

		nfts := make([]NFT, len(listNFTId))
		for i := 0; i < len(listNFTId); i++ {
			row := csvAll[i]
			// accessor sicuro per colonne mancanti
			col := func(k int) string {
				if k >= 0 && k < len(row) {
					return strings.TrimSpace(row[k])
				}
				return ""
			}

			nfts[i] = NFT{
				// campi CSV
				Index:             col(0),
				Name:              col(1),
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

				// campi interni
				TokenID:            listNFTId[i],
				AssignedNodesToken: AssignNFTToNodes(listNFTId[i], iDnew, 2),
			}
		}

		/*

			nfts := make([]NFT, len(listNFTId))
			for i = 0; i < len(listNFTId); i++ {
				//fmt.Printf("Assegnazione NFT %x\n", listNFTId[i])
				nfts[i] = NFT{
					TokenID:            listNFTId[i],
					AssignedNodesToken: AssignNFTToNodes(listNFTId[i], iDnew, 2),
				}

			}
		*/

		// Lista esplicita dei nodi da controllare
		//targets := []string{"node7", "node9"}

		for _, h := range parts {
			if err := waitReady(h, 12*time.Second); err != nil {
				log.Fatalf("❌ Nodo %s non pronto: %v", h, err) // fermati se uno non è pronto
			}
		}

		fmt.Printf("NFT %d: %x,%s,%s\n", 0, nfts[1].TokenID, DecodeID(nfts[1].TokenID), nfts[1].Volume_USD)                              // ID del primo NFT
		fmt.Printf("Assigned bytes Nodes %d: %x, nodo id: %s\n", 1, nfts[1].AssignedNodesToken, DecodeID(nfts[1].AssignedNodesToken[0])) // [NodeID1 NodeID2]

		//-------------Salvatggio degli NFT sugli appositi Nodi-------------------------------------------------------------------//

		fmt.Println("Salvataggio degli NFT assegnati ai nodi...")

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

	} else {

		var nodes []string
		var TokenNodo []byte
		var Bucket [][]byte
		var BucketSort [][]byte

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

		//-------------------I container si mettono in ascolto qui-------------------//
		if err := runGRPCServer(); err != nil {
			log.Fatal(err)
		}

	}

}
