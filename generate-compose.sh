#!/bin/bash

N=10 # oppure leggi dal dataset


echo "services:" >> docker-compose.yml

for i in $(seq 1 $N); do
  cat <<EOL >> docker-compose.yml
  node$i:
    build: .
    environment:
      - NODE_ID=node$i
    ports:
      - "$((8000 + i)):8000"
    networks:
      - kadnet

EOL
done

echo "networks:" >> docker-compose.yml
echo "  kadnet:" >> docker-compose.yml
