package main

import( "encoding/csv"
        "fmt"
        "crypto/sha256"
    "os"
    "log"
    )

type NFT struct {
    ID     string
    Name   string
    Owner  string
}

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

/*

func assignNFTToNode(nodeID string, nftList []string) []ID{
    // Questa funzione dovrebbe assegnare una lista di NFT a un nodo specifico
    if len(nftList) == 0 && nodeID == "" {
        fmt.Println("Nessun NFT da assegnare O ID nodo vuoto")
        return nil
    }

    result := make([]ID, len(nftList))
    for i := 0;i < len(nftList); i++ {
        result[i] =XOR(nftList[i],nodeID)
    }

    //ordina per distanza e prendo i primi 60
    sort.Slice(result, func(i, j int) bool {
        return result[i].LessThan(result[j])
    })

    if len(result) > 60 {
        result = result[:60]
    }

    fmt.Println("Assegnazione NFTs al nodo completata")
    return result
}

*/