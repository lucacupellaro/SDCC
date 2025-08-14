package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type MenuChoice int

const (
	MenuListNodes MenuChoice = iota + 1
	MenuShowBucket
	MenuUseNode
	MenuSearchNFT
	MenuShowEdges
	MenuQuit
)

func ShowWelcomeMenu() MenuChoice {
	clear := func() {
		// best-effort "clear screen" cross-platform (non critico)
		fmt.Print("\033[2J\033[H")
	}

	clear()
	fmt.Println(`
╔══════════════════════════════════════════════╗
║        Kademlia NFT – Console Control        ║
╚══════════════════════════════════════════════╝
Benvenuto! Seleziona un'operazione:

  1) Elenca nodi
  2) Mostra k-bucket di un nodo
  3) Seleziona nodo corrente
  4) Cerca NFT a partire dal nodo corrente
  5) Mostra collegamenti (edges)
  6) Esci
`)

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("Scegli [1-6]: ")
		line, _ := reader.ReadString('\n')
		line = strings.TrimSpace(line)
		switch line {
		case "1":
			return MenuListNodes
		case "2":
			return MenuShowBucket
		case "3":
			return MenuUseNode
		case "4":
			return MenuSearchNFT
		case "5":
			return MenuShowEdges
		case "6", "q", "Q", "exit", "quit":
			return MenuQuit
		default:
			fmt.Println("Scelta non valida, riprova.")
		}
	}
}
