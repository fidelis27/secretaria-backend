# Secretaria Backend

Backend Go isolado em um modulo proprio para manter a possibilidade de
extracao futura para outro repositorio sem reorganizar o restante do monorepo.

## Comandos

Na raiz do repositorio:

```bash
go -C modules/backend test ./...
go -C modules/backend run ./cmd/server
go -C modules/backend run ./cmd/migrate
```

Ou diretamente neste modulo:

```bash
cd modules/backend
go test ./...
go run ./cmd/server
go run ./cmd/migrate
```

## Configuracao

O servidor le `PORT`, `DB_HOST`, `DB_PORT`, `DB_NAME`, `DB_USER`,
`DB_PASSWORD` e `DB_TLS`. A inicializacao abre a conexao MariaDB e valida o
banco com `PingContext`. O endpoint `GET /health` retorna `200` quando o banco
esta disponivel e `503` quando a verificacao falha.

As migrations versionadas sao aplicadas com `go run ./cmd/migrate`. O comando
cria `schema_migrations`, aplica os arquivos SQL em ordem e ignora versoes ja
registradas. No Render, execute-o no Shell do servico depois de configurar as
variaveis `DB_*`.

## Identidade demo

Com excecao de `GET /health` e `POST /users`, as rotas exigem o header
`x-demo-user` com o ID de um usuario ativo persistido no banco:

```bash
curl -H "x-demo-user: <user-id>" http://localhost:3333/institutions
```

Sem o header a API retorna `401`. Usuarios inexistentes ou inativos recebem
`403`. Esse mecanismo e somente para demonstracao local.

## Eventos

O WebSocket `GET /events` exige o mesmo `x-demo-user` e transmite eventos com
`eventId`, `type`, `version`, `source`, `correlationId`, `occurredAt` e
`payload`. Eventos sao persistidos em `audit_events` antes da distribuicao.

## Guia local

O passo a passo para iniciar XAMPP/MariaDB, aplicar migrations, acessar o
phpMyAdmin e subir a API esta em [`docs/local-development.md`](../../docs/local-development.md).
