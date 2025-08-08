package main

import( "encoding/csv"
        "fmt"
        "crypto/sha256"
    "os"
    "log"
    "sort"
    )


type ID [20]byte //160 bit ID


func readCsv(path string) []string {

   

    //fmt.Printf("file: %s\n", path)

    file, err := os.Open(path)
    if err!=nil{
        fmt.Printf("Errore nell'aprire il file CSV: %s\n", err)
        log.Fatal(err)
    }

    defer file.Close()

    reader:= csv.NewReader(file)
    

    records, err := reader.ReadAll()
    if err!=nil{
        fmt.Printf("Errore nella lettura del file CSV: %s\n", err)
        log.Fatal(err)
    }

    
    col1 := make([]string, 0, len(records))
    for i := 1; i < len(records); i++ { // <-- parte da 1
        row := records[i]
        if len(row) < 2 {
            fmt.Printf("Riga %d ha meno di 2 colonne\n", i)
            continue
        }
        col1 = append(col1, row[1])
}


    fmt.Println("CSV file read successfully")
    return col1
}



func NewIDFromToken(tokenID string) ID {
    hash := sha256.Sum256([]byte(tokenID))
    var id ID
    copy(id[:], hash[:20])
    return id
}


func XOR (a,b ID) ID {
    var result ID
    for i := 0; i < len(a); i++ {
        result[i] = a[i] ^ b[i]
    }
    return result
}

// Ordina per distanza: a < b ?
func (a ID) LessThan(b ID) bool {
	for i := 0; i < len(a); i++ {
		if a[i] < b[i] { return true }
		if a[i] > b[i] { return false }
	}
	return false
}


//Generate a list of IDs from a list of tokens or Nodes
func generateID(list []string) []ID {
    if len(list) == 0 {
        fmt.Println("Lista vuota, generazione ID di default")
        return nil
    }

    ids := make([]ID, len(list)) // slice di ID

    for i, tokenStr := range list {
        ids[i] = NewIDFromToken(tokenStr)
    }

    fmt.Println("Generazione ID completata")
    return ids
}




// Restituisce i k nodeID più vicini alla chiave dell'NFT
func AssignNFTToNodes(key ID, nodes []ID, k int) []ID {
	if k <= 0 || len(nodes) == 0 {
		return nil
	}
	if k > len(nodes) {
		k = len(nodes)
	}

	type pair struct {
		id   ID
		dist ID
	}

	pairs := make([]pair, len(nodes))
	for i, nid := range nodes {
		pairs[i] = pair{id: nid, dist: XOR(key, nid)}
	}

	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].dist.LessThan(pairs[j].dist)
	})

	out := make([]ID, k)
	for i := 0; i < k; i++ {
		out[i] = pairs[i].id
	}
	return out
}

/*
func VerifyTopK(nfts []NFT, nodes []ID, k int) error {
	for  nft := range nfts {
		// distanza dei nodi selezionati
		sel := append([]ID(nil), nft.AssignedNodes...)
		sort.Slice(sel, func(a, b int) bool {
			return XOR(nft.TokenID, sel[a]).LessThan(XOR(nft.TokenID, sel[b]))
		})
		// k-esima (max tra i selezionati)
		thr := XOR(nft.TokenID, sel[len(sel)-1])

		// se esiste un nodo con distanza più piccola della soglia, fail
		for _, nid := range nodes {
			d := XOR(nft.TokenID, nid)
			// d < thr ?
			if d.LessThan(thr) {
				// ma nid non è tra i selezionati? (set check)
				inSelected := false
				for _, s := range nft.AssignedNodes {
					if s == nid { inSelected = true; break }
				}
				if !inSelected {
					return fmt.Errorf("NFT %x: nodo %x ha distanza più piccola di %x ma non è stato selezionato",
						nft.TokenID, nid, thr)
				}
			}
		}
	}
	return nil
}

*/
