# Secretaria Backend

API Go da Secretaria Escolar. Este repositorio contem o backend e suas migrations;
os frontends ficam no repositorio `mfe-communication`.

## Comandos

```bash
go test ./...
go run ./cmd/server
go run ./cmd/migrate
```

## Configuracao

O servidor le `PORT`, `DB_HOST`, `DB_PORT`, `DB_NAME`, `DB_USER`,
`DB_PASSWORD`, `DB_TLS`, `KEYCLOAK_ISSUER` e `KEYCLOAK_CLIENT_ID`. Configure o
issuer OIDC do realm Keycloak e o client usado pela API. A inicializacao valida
o issuer e baixa a configuracao JWKS; `GET /health` retorna `200` quando o banco
esta disponivel e `503` quando a verificacao falha.

As migrations versionadas sao aplicadas com `go run ./cmd/migrate`. O comando
cria `schema_migrations`, aplica os arquivos SQL em ordem e ignora versoes ja
registradas. No Render, execute-o no Shell do servico depois de configurar as
variaveis `DB_*`.

## Autenticacao e autorizacao

Todas as rotas, exceto `GET /health`, exigem access token Keycloak no header
Bearer. O backend valida assinatura, issuer, audience e expiracao via OIDC/JWKS,
exige e-mail verificado e associa o e-mail a um usuario ativo no banco. O papel
`super_admin` precisa existir tanto na role do realm quanto no cadastro local.

```bash
curl -H "Authorization: Bearer <access-token>" http://localhost:3333/institutions
```

Sem token valido a API retorna `401`; usuarios sem cadastro ativo recebem `403`.

## Eventos

O WebSocket `GET /events` exige token no subprotocolo `bearer.<access-token>` e transmite eventos com
`eventId`, `type`, `version`, `source`, `correlationId`, `occurredAt` e
`payload`. Eventos sao persistidos em `audit_events` antes da distribuicao.

## Desenvolvimento local

Inicie MariaDB, configure as variaveis `DB_*` e `KEYCLOAK_*`, aplique as
migrations e execute `go run ./cmd/server`. O guia do monorepo
`mfe-communication` documenta a inicializacao coordenada dos MFEs e desta API.
