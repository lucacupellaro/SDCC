package ui

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	pb "kademlia-nft/proto/kad"

	"kademlia-nft/logica"

	"math/big"
	"os"
	"os/exec"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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

type KademliaServer struct {
	pb.UnimplementedKademliaServer
}

func ShowWelcomeMenu() MenuChoice {
	fmt.Print("\033[2J\033[H") // clear screen
	fmt.Println(`
╔══════════════════════════════════════════════╗
║        Kademlia NFT – Console Control        ║
╚══════════════════════════════════════════════╝
Benvenuto! Seleziona un'operazione:

  1) Elenca nodi
  2) Mostra k-bucket di un nodo
  3) Cerca un NFT  
  4) Aggiungi un NFT
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

// Restituisce i servizi Compose attivi (node1, node2, ...) del progetto.
func ListActiveComposeServices(project string) ([]string, error) {
	// Filtra per progetto per non mischiare altri container.
	// --format {{.Names}} per ottenere i nomi dei container es.: kademlia-nft-node1-1
	cmd := exec.Command("docker", "ps",
		"--filter", "label=com.docker.compose.project="+project,
		"--format", "{{.Names}}",
	)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	services := make([]string, 0, len(lines))
	for _, name := range lines {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		// Nome Compose tipico: <project>-<service>-<index>
		parts := strings.Split(name, "-")
		if len(parts) >= 3 {
			services = append(services, parts[len(parts)-2]) // prende <service> (es. "node1")
		}
	}
	return dedupe(services), nil
}

func dedupe(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

func resolveStartHostPort(name string) (string, error) {
	name = strings.TrimSpace(strings.ToLower(name))
	// supporta sia "node3" sia "nodo3"
	if strings.HasPrefix(name, "nodo") {
		name = "node" + name[len("nodo"):]
	}
	var n int
	if _, err := fmt.Sscanf(name, "node%d", &n); err != nil || n < 1 || n > 11 {
		return "", fmt.Errorf("nome nodo non valido: %q", name)
	}
	// La CLI corre su HOST → usa la porta mappata localhost:800N
	return fmt.Sprintf("localhost:%d", 8000+n), nil
}

/*
	func LookupNFTOnNodeByName(nodeName, nftName string) error {
		hostPort, err := resolveStartHostPort(nodeName)
		if err != nil {
			return err
		}

		fmt.Printf("🔎 Cerco '%s' su %s\n", nftName, hostPort)

		// inviamo il NOME in chiaro: il server farà pad+hex per costruire <id>.json
		key := []byte(nftName)

		conn, err := grpc.Dial(hostPort, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return fmt.Errorf("dial fallito %s: %w", hostPort, err)
		}
		defer conn.Close()

		client := pb.NewKademliaClient(conn)
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		resp, err := client.LookupNFT(ctx, &pb.LookupNFTReq{
			FromId: "CLI",
			Key:    &pb.Key{Key: key},
		})
		if err != nil {
			return fmt.Errorf("RPC fallita: %w", err)
		}

		if !resp.GetFound() {
			fmt.Println("✖ NFT non trovato su questo nodo")
			return nil
		}

		// Stampa il contenuto JSON
		fmt.Printf("✓ Trovato su nodo %s\n", resp.GetHolder().GetId())
		fmt.Printf("Contenuto JSON:\n%s\n", string(resp.GetValue().GetBytes()))
		return nil
	}
*/
func LookupNFTOnNodeByName(startNode, nftName string, maxHops int) error {
	if maxHops <= 0 {
		maxHops = 15
	}

	nftID20 := logica.NewIDFromToken(nftName, 20)
	visited := make(map[string]bool)
	current := startNode

	for hop := 0; hop < maxHops; hop++ {
		if visited[current] {
			// già visto: non ha senso riprovarlo
			break
		}
		visited[current] = true

		hostPort, err := resolveStartHostPort(current)
		if err != nil {
			return fmt.Errorf("risoluzione %q fallita: %w", current, err)
		}

		fmt.Printf("🔎 Hop %d: cerco '%s' su %s (%s)\n", hop+1, nftName, current, hostPort)

		conn, err := grpc.Dial(hostPort, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return fmt.Errorf("dial fallito %s: %w", hostPort, err)
		}
		client := pb.NewKademliaClient(conn)

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		resp, rpcErr := client.LookupNFT(ctx, &pb.LookupNFTReq{
			FromId: "CLI",
			Key:    &pb.Key{Key: []byte(nftName)}, // nome in chiaro; il server fa pad+hex
		})
		cancel()
		_ = conn.Close()

		if rpcErr != nil {
			return fmt.Errorf("RPC fallita su %s: %w", current, rpcErr)
		}

		if resp.GetFound() {
			fmt.Printf("✅ Trovato su nodo %s\n", resp.GetHolder().GetId())
			fmt.Printf("Contenuto JSON:\n%s\n", string(resp.GetValue().GetBytes()))
			return nil
		}

		nearest := resp.GetNearest()
		if len(nearest) == 0 {
			fmt.Println("✖ NFT non trovato e nessun nodo vicino restituito — arresto.")
			return nil
		}

		// Estrai gli ID utili e filtra già i visitati
		candidates := make([]string, 0, len(nearest))
		fmt.Println("… nodi vicini suggeriti:")
		for _, n := range nearest {
			id := n.GetId()
			if id == "" {
				id = n.GetHost()
			}
			if id == "" {
				continue
			}
			fmt.Printf("   - %s (%s:%d)\n", id, n.GetHost(), n.GetPort())
			if !visited[id] {
				candidates = append(candidates, id)
			}
		}

		if len(candidates) == 0 {
			fmt.Println("✖ Nessun vicino non visitato disponibile — arresto.")
			return nil
		}

		best, err := sceltaNodoPiuVicino(nftID20, candidates)
		if err != nil {
			fmt.Printf("⚠️  Impossibile scegliere il nodo più vicino: %v — prendo il primo candidato.\n", err)
			best = candidates[0]
		}

		fmt.Printf("➡️  Prossimo nodo scelto: %s\n", best)
		current = best
	}

	fmt.Printf("⛔ Max hop (%d) raggiunto senza trovare '%s'.\n", maxHops, nftName)
	return nil
}

// sceltaNodoPiuVicino: XOR distance minima tra nftID20 e ogni nodo (ID a 20 byte).
func sceltaNodoPiuVicino(nftID20 []byte, nodiVicini []string) (string, error) {
	var bestNode string
	var bestDist *big.Int

	for _, idStr := range nodiVicini {
		// Ricostruisco l'ID a 20 byte come fai ovunque (NewIDFromToken)
		nidBytes := logica.NewIDFromToken(idStr, 20)

		// XOR byte-wise
		distBytes := make([]byte, len(nftID20))
		for i := range nftID20 {
			distBytes[i] = nftID20[i] ^ nidBytes[i]
		}

		distInt := new(big.Int).SetBytes(distBytes)
		if bestDist == nil || distInt.Cmp(bestDist) < 0 {
			bestDist = distInt
			bestNode = idStr
		}
	}

	if bestNode == "" {
		return "", fmt.Errorf("nessun nodo valido trovato")
	}
	return bestNode, nil
}
