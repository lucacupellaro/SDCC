package main


import (
	"fmt"
	"os"
	  "strings" 
)

func main() {

	var listNFT []string
	var listNFTId []ID

	// Prende l'ID del nodo dall'ambiente
	nodeID := os.Getenv("NODE_ID")
	if nodeID == "" {
		nodeID = "default"
	}

	fmt.Println("Avviato nodo:", nodeID)

	isSeeder := os.Getenv("SEED") == "true"

	if isSeeder {


		fmt.Printf("sono il seeder,\nReading CSV file...\n")

		listNFT=readCsv("csv/NFT_Top_Collections.csv")


		fmt.Printf("NFT letti: %d\n", len(listNFT))

		fmt.Printf("NFT letto da csv: %s\n", listNFT[0]) 


		// Genera gli ID per la lista di NFT
		fmt.Printf("Generating IDs for NFTs...\n")

		listNFTId = generateID(listNFT)

		fmt.Printf("NFT id %s:\n", listNFTId[0])


		// recuper gli ID dei container
		rawNodes := os.Getenv("NODES")
		if rawNodes == "" {
			fmt.Println("Nessun nodo in NODES")
			return
		}

		
		
		parts := strings.Split(rawNodes, ",")
		iDnew := make([]ID,len(parts))
		for i, p := range parts {
			fmt.Println("Nodo trovato:", p)
			iDnew[i] = NewIDFromToken(p)
			//fmt.Printf("ID sha256 truncato del nodo: %x\n", iDnew[i])
		}

		 //fissato nft quindi per ogni nft , crei unafunzione che per ogni nft scorre tutti e e 10 gli i d dei nodi e li assegna ai 2/3 pi vicini)

	}

	




}


	// Qui andrai a:
	// - Inizializzare la tabella di routing
	// - Caricare NFT associati al nodo (per ora 1 per nodo)
	// - Avviare server (TCP/HTTP) per comunicare con altri nodi
	// - Implementare messaggi PING, STORE, FIND_NODE, ecc.
