package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

type NFT struct {
	TokenID            []byte
	Name               string
	AssignedNodesToken [][]byte
}

func main() {

	var listNFT []string
	var listNFTId [][]byte
	var i int

	// Prende l'ID del nodo dall'ambiente
	nodeID := os.Getenv("NODE_ID")
	if nodeID == "" {
		nodeID = "default"
	}

	fmt.Println("Avviato nodo:", nodeID)

	isSeeder := os.Getenv("SEED") == "true"

	if isSeeder {

		// Avvia SEMPRE il server gRPC

		// sul seeder avvialo in background
		go func() {
			if err := runGRPCServer(); err != nil {
				log.Fatal(err)
			}
		}()

		fmt.Printf("sono il seeder,\nReading CSV file...\n")

		listNFT = readCsv("csv/NFT_Top_Collections.csv")

		fmt.Printf("NFT letti: %d\n", len(listNFT))

		fmt.Printf("NFT letto da csv: %s\n", listNFT[0])

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
		for i = 0; i < len(listNFTId); i++ {
			//fmt.Printf("Assegnazione NFT %x\n", listNFTId[i])
			nfts[i] = NFT{
				TokenID:            listNFTId[i],
				AssignedNodesToken: AssignNFTToNodes(listNFTId[i], iDnew, 2),
			}

		}

		// Lista esplicita dei nodi da controllare
		targets := []string{"node7", "node9"}

		for _, h := range targets {
			if err := waitReady(h, 12*time.Second); err != nil {
				log.Fatalf("❌ Nodo %s non pronto: %v", h, err) // fermati se uno non è pronto
			}
		}

		fmt.Printf("NFT %d: %x,%s\n", 1, nfts[1].TokenID, DecodeID(nfts[1].TokenID))                                                     // ID del primo NFT
		fmt.Printf("Assigned bytes Nodes %d: %x, nodo id: %s\n", 1, nfts[1].AssignedNodesToken, DecodeID(nfts[1].AssignedNodesToken[0])) // [NodeID1 NodeID2]

		/*
					// Stampa i risultati
					for i := 0; i < len(listNFTId); i++ {
						fmt.Printf("NFT %d: %x\n", i, nfts[i].TokenID)       // ID del primo NFT
						fmt.Printf("Assigned Nodes %d: %x\n", i, nfts[i].AssignedNodesToken) // [NodeID1 NodeID2]
					}
			/*


						if err := VerifyTopK(nfts, iDnew, 2); err != nil {
							fmt.Println("VERIFICA FALLITA:", err)
						} else {
							fmt.Println("Verifica OK: ogni NFT è assegnato ai 2 nodi più vicini.")
						}
		*/

		//-------------Salvatggio degli NFT sull'apposito Nodo----//
		fmt.Println("Salvataggio degli NFT assegnati ai nodi...")
		// idToURL: mappa NodeID -> URL (riempila dai tuoi ENV / compose)

		err := StoreNFTToNodes("BAYC#123", "BAYC #123", []string{"node7", "node9"}, 24*3600)
		if err != nil {
			fmt.Println("Errore:", err)
		}

	} else {
		// sui non-seeder blocca qui
		if err := runGRPCServer(); err != nil {
			log.Fatal(err)
		}
		return
	}

}
