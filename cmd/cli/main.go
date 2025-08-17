package main

import (
	"fmt"
	"kademlia-nft/internal/ui"
	"log"
)

func main() {

	choice := ui.ShowWelcomeMenu()
	fmt.Println("Hai scelto:", choice)

	if choice == 3 {
		var nodi []string
		nodi, err := ui.ListActiveComposeServices("kademlia-nft")
		if err != nil {
			log.Fatal("Errore recupero nodi:", err)
		}
		fmt.Println("Container attivi:")
		for _, n := range nodi {
			fmt.Println(" -", n)
		}
		/*
			fmt.Println("Da quale nodo vuoi far partire la simulazione?")
			fmt.Println("Per selezionare un nodo, usa il comando 'use <nome-nodo>'")
			nodoScelto := bufio.NewReader(os.Stdin)
			line, _ := nodoScelto.ReadString('\n')
			line = strings.TrimSpace(line)

			fmt.Println("Hai scelto il nodo:", line)

			fmt.Println("Quale Nft vuoi cercare?")
			nftScelto := bufio.NewReader(os.Stdin)
			line1, _ := nftScelto.ReadString('\n')
			line1 = strings.TrimSpace(line1)

			fmt.Println("Hai scelto il NFT:", line1)

			key := NewIDFromToken(line1, 20)
			//fmt.Printf("%x", key)
		*/

		//------------------------Inizia la ricerca dell'NFT-------------------------------------------//
		node := "nodo7" // o "node3"
		name := "Kreechures"

		if err := ui.LookupNFTOnNodeByName(node, name, 7); err != nil {
			fmt.Println("Errore:", err)
		}

	}

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
