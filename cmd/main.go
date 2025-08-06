package main

import (
	"fmt"
	"os"
)

func main() {
	// Prende l'ID del nodo dall'ambiente
	nodeID := os.Getenv("NODE_ID")
	if nodeID == "" {
		nodeID = "default"
	}

	fmt.Println("Avviato nodo:", nodeID)

	// Qui andrai a:
	// - Inizializzare la tabella di routing
	// - Caricare NFT associati al nodo (per ora 1 per nodo)
	// - Avviare server (TCP/HTTP) per comunicare con altri nodi
	// - Implementare messaggi PING, STORE, FIND_NODE, ecc.

	

}
