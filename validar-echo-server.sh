#!/bin/bash

MSG="hello"
COMPOSE_FILE="${COMPOSE_FILE:-docker-compose-dev.yaml}"

# Busco el nombre del proyecto en el archivo del compose
PROJECT_NAME="$(awk '/^[[:space:]]*name:[[:space:]]*/{print $2; exit}' "$COMPOSE_FILE" 2>/dev/null || echo tp0)"
NETWORK="${PROJECT_NAME}_testing_net"

# Busca en el listado de contenedores en ejecución uno que se llame exactamente 'server'
# en caso de no encontrarlo sale y no intenta de vuelta
if ! docker ps --format '{{.Names}}' | grep -qx 'server'; then
  echo "action: test_echo_server | result: fail"
  exit 1
fi

# Ejecuta python dentro de server e intenta leer el archivo config.ini para buscar el puerto.
# Si existe SERVER_PORT en el entorno lo toma, si no toma DEFAULT.SERVER_PORT de config.ini y 
# si tampoco existe toma 12345 por defecto
PORT_OUT="$(docker exec server python3 - <<'PY' 2>/dev/null || true
import os
from configparser import ConfigParser
c = ConfigParser(os.environ)
c.read("config.ini")
print(os.getenv("SERVER_PORT", c["DEFAULT"].get("SERVER_PORT", "12345")))
PY
)"

PORT="${PORT_OUT:-12345}"
# Si el puerto tuviera un formato inválido (no numérico) toma el 12345 por defecto
case "$PORT" in
  ''|*[!0-9]*) PORT="12345";;
esac

# Crea un contenedor temporal y lo conecta a la red del compose para resolver 'server'.
# Usa alpine y ejecuta un shell. Envia por stdin a netcat conectado con 'server' en $PORT el mensaje sin salto de línea
RESPONSE="$(docker run --rm --network "$NETWORK" alpine:3.19 sh -c "echo -n '$MSG' | nc server $PORT" 2>/dev/null || true)"

if [ "$RESPONSE" = "$MSG" ]; then
  echo "action: test_echo_server | result: success"
else
  echo "action: test_echo_server | result: fail"
fi