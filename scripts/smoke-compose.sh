#!/usr/bin/env sh
set -eu

cleanup() {
  docker compose down
}
trap cleanup EXIT

docker compose up --build --detach api

url="http://127.0.0.1:${HOST_PORT:-3333}/health"
attempt=0
while [ "$attempt" -lt 30 ]; do
  if curl --fail --silent "$url"; then
    printf '\nCompose smoke test passed: GET %s returned 200\n' "$url"
    exit 0
  fi
  attempt=$((attempt + 1))
  sleep 2
done

docker compose logs api migrate db
printf 'Compose smoke test failed: GET %s did not return 200\n' "$url" >&2
exit 1
