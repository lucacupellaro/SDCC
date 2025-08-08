#!/usr/bin/env bash
set -euo pipefail

N=11

# Costruisci la lista "node2:8000,node3:8000,...,nodeN:8000"
NODES=""
for i in $(seq 2 "$N"); do
  NODES+="node$i,"
done
NODES="${NODES%,}"  # togli la virgola finale

# Sovrascrivi il file docker-compose.yml
cat > docker-compose.yml <<EOF
services:
EOF

for i in $(seq 1 "$N"); do
  cat >> docker-compose.yml <<EOF
  node$i:
    build: .
    environment:
      - NODE_ID=node$i
EOF

  # Solo il primo nodo ha SEED e la lista dei peer
  if [ "$i" -eq 1 ]; then
    cat >> docker-compose.yml <<EOF
      - SEED=true
      - NODES=$NODES
EOF
  fi

  cat >> docker-compose.yml <<EOF
    ports:
      - "$((8000 + i)):8000"
    networks:
      - kadnet

EOF
done

cat >> docker-compose.yml <<EOF
networks:
  kadnet:
EOF
