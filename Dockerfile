# Usa l'immagine ufficiale di Go come base
FROM golang:1.21-alpine

# Crea la directory di lavoro dentro il container
WORKDIR /app

# Copia i file del progetto nel container
COPY . .

# Recupera le dipendenze
RUN go mod tidy

# Compila il main
RUN go build -o node ./cmd/main.go

# Avvia il binario
CMD ["./node"]
