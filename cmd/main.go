package main


import (
	"fmt"
	"os"
)

func main() {

	listNFT string[]

	// Prende l'ID del nodo dall'ambiente
	nodeID := os.Getenv("NODE_ID")
	if nodeID == "" {
		nodeID = "default"
	}

	fmt.Println("Avviato nodo:", nodeID)

	fmt.Printf("Reading CSV file...\n")

	listNFT=readCsv("csv/NFT_Top_Collections.csv")


	listNFTId=generateID(listNFT)



	IDnew := NewIDFromToken(nodeID)
	fmt.Printf("ID sha256 truncato del nodo: %s\n", IDnew)





}


	// Qui andrai a:
	// - Inizializzare la tabella di routing
	// - Caricare NFT associati al nodo (per ora 1 per nodo)
	// - Avviare server (TCP/HTTP) per comunicare con altri nodi
	// - Implementare messaggi PING, STORE, FIND_NODE, ecc.
