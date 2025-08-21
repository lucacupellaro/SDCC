package ui

import (
	"bufio"
	"bytes"
	"context"
	"encoding/hex"
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

func ShowWelcomeMenu() MenuChoice {
	fmt.Print("\033[2J\033[H") // clear screen
	fmt.Println(`
╔══════════════════════════════════════════════╗
║        Kademlia NFT – Console Control        ║
╚══════════════════════════════════════════════╝
Benvenuto! Seleziona un'operazione:

  1) Elenca nodi
  2) Ping (X->Y)
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

func RPCGetKBucket(nodeAddr string) ([]string, error) {

	add, err := resolveStartHostPort(nodeAddr)
	fmt.Printf("🔍 Recupero KBucket di %s\n", add)
	if err != nil {
		return nil, fmt.Errorf("risoluzione %q fallita: %w", nodeAddr, err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, add,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %v", add, err)
	}
	defer conn.Close()

	client := pb.NewKademliaClient(conn)
	resp, err := client.GetKBucket(ctx, &pb.GetKBucketReq{RequesterId: "cli"})
	if err != nil {
		return nil, fmt.Errorf("rpc GetKBucket: %v", err)
	}

	// converto []*pb.Node in []string
	var res []string
	for _, n := range resp.Nodes {
		res = append(res, n.Id)
	}

	return res, nil
}

/*
func PingNode(startNode, targetNode string) {
	current := startNode
	targetID := logica.NewIDFromToken(targetNode, 20) // per distanza XOR
	visited := map[string]bool{}
	maxHops := 20
	var nodeVisited []string

	for hop := 0; hop < maxHops; hop++ {
		fmt.Printf("🔍 Inizio PING da %s a %s (hop %d)\n", current, targetNode, hop+1)

		// 1) KBucket del nodo corrente
		nodi, err := RPCGetKBucket(current)
		if err != nil {
			log.Fatal("Errore RPCGetKBucket:", err)
		}

		// Se il server ti restituisce esadecimale, normalizza a "nodeX".
		nodi = normalizeIDs(nodi)

		// 2) Trovato direttamente?
		found := false
		for _, n := range nodi {
			if n == targetNode {
				found = true
				break
			}
		}
		if found {
			fmt.Printf("🔍 Nodo %s trovato in KBucket di %s\n", targetNode, current)
			fmt.Printf("🔍 Stiamo mandando una richista di Ping\n")
			if err := SendPing(current, targetNode); err != nil {
				fmt.Printf("⚠️  Errore durante l'invio del Ping: %v\n", err)
			}
			return
		}

		// 3) Scegli il prossimo vicino al target evitando i già visitati
		visited[current] = true
		candidates := make([]string, 0, len(nodi))
		for _, n := range nodi {
			if !visited[n] {
				candidates = append(candidates, n)
			}
		}
		if len(candidates) == 0 {
			fmt.Println("✖ Nessun vicino non visitato — arresto.")
			return
		}

		best, err := sceltaNodoPiuVicino(targetID, candidates)
		if err != nil {
			fmt.Printf("⚠️  Impossibile scegliere il nodo più vicino: %v\n", err)
			return
		}
		fmt.Printf("➡️  Prossimo nodo scelto: %s\n", best)
		current = best
	}

	fmt.Println("⛔ Max hop raggiunto senza raggiungere il target.")
}
*/

func xorDist(a20 []byte, b20 []byte) *big.Int {
	nb := make([]byte, len(a20))
	for i := range a20 {
		nb[i] = a20[i] ^ b20[i]
	}
	return new(big.Int).SetBytes(nb)
}

func PingNode(startNode, targetNode string) {
	targetID := logica.NewIDFromToken(targetNode, 20)
	visited := map[string]bool{}
	candidates := map[string]bool{} // insieme senza duplicati
	addCand := func(list []string) {
		for _, n := range list {
			if n == "" {
				continue
			}
			candidates[n] = true
		}
	}

	// seed: parti dal nodo iniziale e dai suoi vicini
	addCand([]string{startNode})

	bestDist := (*big.Int)(nil)
	stagnate := 0
	maxHops := 30

	for hop := 0; hop < maxHops; hop++ {
		// scegli il candidato non visitato più vicino al target
		var next string
		var nextD *big.Int
		for id := range candidates {
			if visited[id] {
				continue
			}
			d := xorDist(targetID, logica.NewIDFromToken(id, 20))
			if next == "" || d.Cmp(nextD) < 0 {
				next = id
				nextD = d
			}
		}
		if next == "" {
			fmt.Println("✖ Nessun vicino non visitato — arresto.")
			return
		}

		fmt.Printf("🔍 Inizio PING da %s a %s (hop %d)\n", next, targetNode, hop+1)
		visited[next] = true

		// prendi KBucket del candidato
		kb, err := RPCGetKBucket(next)
		if err != nil {
			fmt.Printf("⚠️  GetKBucket(%s) fallita: %v\n", next, err)
			continue
		}
		kb = normalizeIDs(kb)
		fmt.Printf("🔎 %s ha %d vicini\n", next, len(kb))

		// target presente?
		for _, n := range kb {
			if n == targetNode {
				fmt.Printf("✅ %s conosce %s — invio Ping…\n", next, targetNode)
				if err := SendPing(next, targetNode); err != nil {
					fmt.Printf("⚠️  Ping fallito: %v\n", err)
				}
				return
			}
		}

		// accumula nuovi candidati
		addCand(kb)

		// controllo progresso
		if bestDist == nil || nextD.Cmp(bestDist) < 0 {
			bestDist = nextD
			stagnate = 0
		} else {
			stagnate++
			if stagnate >= 2 {
				fmt.Println("⛔ Nessun miglioramento di distanza — arresto.")
				return
			}
		}
	}
	fmt.Println("⛔ Max hop raggiunto senza contattare il target.")
}

// Se GetKBucket ti manda esadecimale (20 byte = 40 char), decodifichiamo in "nodeX".
func normalizeIDs(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		if len(s) == 40 { // sembra hex (20 byte)
			if b, err := hex.DecodeString(s); err == nil {
				s = string(bytes.TrimRight(b, "\x00"))
			}
		}
		out = append(out, s)
	}
	return out
}

func SendPing(fromID, targetName string) error {
	// riusa la tua funzione che risolve l’endpoint del nodo (host:port)
	addr, err := resolveStartHostPort(targetName) // es: "localhost:8004"
	if err != nil {
		return err
	}

	// connessione con timeout e block (meglio feedback chiaro sulle reachability)
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(ctx, addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
		grpc.WithReturnConnectionError(),
	)
	if err != nil {
		return fmt.Errorf("dial %s: %w", addr, err)
	}
	defer conn.Close()

	client := pb.NewKademliaClient(conn)
	resp, err := client.Ping(ctx, &pb.PingReq{
		From: &pb.Node{Id: fromID, Host: fromID, Port: 0}, // meta: Host/Port opzionali
	})
	if err != nil {
		return fmt.Errorf("Ping %s: %w", targetName, err)
	}

	fmt.Printf("PONG da %s (ok=%v, t=%d)\n", resp.GetNodeId(), resp.GetOk(), resp.GetUnixMs())
	// (opzionale) aggiorna la routing table locale di X con Y, perché ha risposto:
	// UpdateBucketLocal(targetName)

	return nil
}
