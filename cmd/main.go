package main


import (
	"fmt"
	"os"
	  "strings" 
)

type NFT struct {
	TokenID ID
	Name    string
	AssignedNodes []ID
}

func main() {

	var listNFT []string
	var listNFTId []ID
	var i int

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

		fmt.Println("Assegnazione dei k nodeID più vicini agli NFT...")

		
		nfts := make([]NFT, len(listNFTId))
		for i = 0; i < len(listNFTId); i++ {
			//fmt.Printf("Assegnazione NFT %x\n", listNFTId[i])
			nfts[i] = NFT{
				TokenID:       listNFTId[i],
				AssignedNodes: AssignNFTToNodes(listNFTId[i], iDnew, 2),
			}
			
		}



		/*
		for i=0;i<len(listNFTId);i++ {
				fmt.Printf("NFT %d: %x\n", i, nfts[i].TokenID)       // ID del primo NFT
				fmt.Printf("Assigned Nodes %d: %x\n", i, nfts[i].AssignedNodes) // [NodeID1 NodeID2]
		}

		
		if err := VerifyTopK(nfts, iDnew, 2); err != nil {
			fmt.Println("VERIFICA FALLITA:", err)
		} else {
			fmt.Println("Verifica OK: ogni NFT è assegnato ai 2 nodi più vicini.")
		}
		*/

		//-------------Salvatggio degli NFT sull'apposito Nodo----//
		fmt.Println("Salvataggio degli NFT assegnati ai nodi...")
		// idToURL: mappa NodeID -> URL (riempila dai tuoi ENV / compose)
		for _, nft := range nfts {
			for _, nid := range nft.AssignedNodes {
				url := idToURL[nid]
				if url == "" { 
					fmt.Printf("URL mancante per nodo %x\n", nid)
					continue 
				}
				if err := StoreOverHTTP(url, nft.TokenID, NFTValue{Name: nft.Name}); err != nil {
					fmt.Printf("STORE fail su %s: %v\n", url, err)
				}
			}
		}


		
	}

	




}



	// Qui andrai a:
	// - Inizializzare la tabella di routing
	// - Caricare NFT associati al nodo (per ora 1 per nodo)
	// - Avviare server (TCP/HTTP) per comunicare con altri nodi
	// - Implementare messaggi PING, STORE, FIND_NODE, ecc.
