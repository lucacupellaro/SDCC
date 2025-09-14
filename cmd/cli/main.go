package main

import (
	"context"
	"fmt"
	"kademlia-nft/internal/ui"
	"log"
	"os"
)

func main() {

	choice := ui.ShowWelcomeMenu()
	fmt.Println("Hai scelto:", choice)

	if choice == 1 {
		fmt.Printf("Hai scelto l'opzione 1\n")
	}

	if choice == 2 {
		fmt.Printf("Hai scelto l'opzione 2. PING\n")

		/*


			fmt.Printf("Da quale nodo vuoi fare il PING?\n")

			nodi, err := ui.ListActiveComposeServices("kademlia-nft")
			if err != nil {
				log.Fatal("Errore recupero nodi:", err)
			}
			fmt.Println("Container attivi:")
			for _, n := range nodi {
				fmt.Println(" -", n)
			}

			fmt.Printf("Verso quale nodo vuoi fare il PING?\n")
			fmt.Println("Container attivi:")
			for _, n := range nodi {
				fmt.Println(" -", n)
			}
		*/

		ui.PingNode("node8", "node5")

	}

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

			//fmt.Printf("%x", key)
		*/

		//------------------------Inizia la ricerca dell'NFT-------------------------------------------//
		node := "nodo6" // o "node3"
		name := "Lift-off Pass"

		if err := ui.LookupNFTOnNodeByName(node, name, 30); err != nil {
			fmt.Println("Errore:", err)
		}

	}
	if choice == 4 {

		/*

			var nodi []string

			fmt.Println("Aggiungi un NFT")
			nodoScelto := bufio.NewReader(os.Stdin)
			line, _ := nodoScelto.ReadString('\n')
			line = strings.TrimSpace(line)
			fmt.Println("Hai scelto il NFT:", line)

			nodi, err := ui.ListActiveComposeServices("kademlia-nft")
			if err != nil {
				log.Fatal("Errore recupero nodi:", err)
			}
			fmt.Println("Container attivi:")
			for _, n := range nodi {
				fmt.Println(" -", n)
			}

			logica.RemoveNode1(&nodi)

			var iDnew [][]byte
			iDnew = make([][]byte, len(nodi))
			for i, p := range nodi {

				iDnew[i] = logica.NewIDFromToken(p, 20)

			}

			key := logica.NewIDFromToken(line, 20) // ID dal Name (come vuoi tu)

			nfts := make([]logica.NFT, 0, 1)
			nfts = append(nfts, logica.NFT{
				Index:             "col(0)",
				Name:              line,
				Volume:            "col(2)",
				Volume_USD:        "col(3)",
				Market_Cap:        "col(4)",
				Market_Cap_USD:    "col(5)",
				Sales:             "col(6)",
				Floor_Price:       "col(7)",
				Floor_Price_USD:   "col(8)",
				Average_Price:     "col(9)",
				Average_Price_USD: "col(10)",
				Owners:            "col(11)",
				Assets:            "col(12)",
				Owner_Asset_Ratio: "col(13)",
				Category:          "col(14)",
				Website:           "col(15)",
				Logo:              "col(16)",

				TokenID:            key,
				AssignedNodesToken: logica.AssignNFTToNodes(key, iDnew, 2),
			})
			var nodiSeletcted []string

			fmt.Printf("sto salvando nft %s nei nodi: %s,%s\n", nfts[0].Name, logica.DecodeID(nfts[0].AssignedNodesToken[0]), logica.DecodeID(nfts[0].AssignedNodesToken[1]))
			nodiSeletcted = append(nodiSeletcted, logica.DecodeID(nfts[0].AssignedNodesToken[0]))
			nodiSeletcted = append(nodiSeletcted, logica.DecodeID(nfts[0].AssignedNodesToken[1]))

			if err := logica.StoreNFTToNodes2(nfts[0], logica.DecodeID(nfts[0].TokenID), nfts[0].Name, nodiSeletcted, 24*3600); err != nil {
				fmt.Println("Errore:", err)

			}


		*/
	}
	if choice == 5 {

		var nodi []string
		var biggerNode string

		fmt.Println("Aggiungo un nuovo nodo")

		nodi, err := ui.ListActiveComposeServices("kademlia-nft")
		if err != nil {
			log.Fatal("Errore recupero nodi:", err)
		}

		biggerNode = ui.BiggerNodes(nodi)

		ctx := context.Background()

		// esempio: aggiunge node12 collegato al seeder node1
		if err := ui.AddNode(ctx, biggerNode, "node1:8000", "8012"); err != nil {
			fmt.Println("Errore:", err)
			os.Exit(1)
		}
	}

}
