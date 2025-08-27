#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 2 ]]; then
  echo "Comando esperado: $0 docker-compose-dev.yaml <n_clients>"
  exit 1
fi

OUTPUT_FILE="$1"
NUM_CLIENTS="$2"

if ! [[ "$NUM_CLIENTS" =~ ^[0-9]+$ ]] || [[ "$NUM_CLIENTS" -lt 1 ]]; then
  echo "Error:  debe ser un entero >= 1"
  exit 2
fi

cat > "$OUTPUT_FILE" <<'END'
name: tp0
services:
  server:
    container_name: server
    image: server:latest
    entrypoint: python3 /main.py
    environment:
      - PYTHONUNBUFFERED=1
    volumes:
      - ./server/config.ini:/config.ini:ro
    networks:
      - testing_net
END

for i in $(seq 1 "$NUM_CLIENTS"); do
  cat >> "$OUTPUT_FILE" <<END
  client${i}:
    container_name: client${i}
    image: client:latest
    entrypoint: /client
    environment:
      - CLI_ID=${i}
      volumes:
      - ./client/config.yaml:/config.yaml:ro
    networks:
      - testing_net
    depends_on:
      - server
END
done

cat >> "$OUTPUT_FILE" <<'END'
networks:
  testing_net:
    ipam:
      driver: default
      config:
        - subnet: 172.25.125.0/24
END

echo "Compose generado en: $OUTPUT_FILE con ${NUM_CLIENTS} cliente(s)."


