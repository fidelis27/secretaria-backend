# Secretaria Backend

Backend Go isolado em um modulo proprio para manter a possibilidade de
extracao futura para outro repositorio sem reorganizar o restante do monorepo.

## Comandos

Na raiz do repositorio:

```bash
go -C modules/backend test ./...
go -C modules/backend run ./cmd/server
```

Ou diretamente neste modulo:

```bash
cd modules/backend
go test ./...
go run ./cmd/server
```

## Configuracao

O servidor le `PORT`, `DB_HOST`, `DB_PORT`, `DB_NAME`, `DB_USER`,
`DB_PASSWORD` e `DB_TLS`. A conexao com MariaDB sera adicionada na proxima
fatia; por enquanto, o modulo expoe apenas o endpoint `GET /health`.