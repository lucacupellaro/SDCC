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

    fmt.Printf("file: %s\n", path)

    file, err := os.Open(path)
    if err!=nil{
        fmt.Printf("Errore nell'aprire il file CSV: %s\n", err)
        log.Fatal(err)
    }

    reader:= csv.NewReader(file)
    

    records, err := reader.ReadAll()
    if err!=nil{
        fmt.Printf("Errore nella lettura del file CSV: %s\n", err)
        log.Fatal(err)
    }

    print(records)

    for i, row :=range records{
        if len(row)<2{
            fmt.Printf("Riga %d ha meno di 2 colonne",i)
            continue
         
        }

        column1 := row[0]
        column2 := row[1]
        fmt.Printf("Riga %d: %s, %s\n", i, column1, column2)
    }

    //fmt.Printf("colonna 1: %s, colonna 2: %s\n", records[0][0], records[0][1])

    file.Close()
    fmt.Println("CSV file read successfully")

    return column1
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