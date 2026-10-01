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
`DB_PASSWORD`, `DB_TLS`, `OIDC_ISSUER` e `OIDC_AUDIENCE`. Para Supabase Auth,
configure `OIDC_ISSUER` com `https://<project-ref>.supabase.co/auth/v1` e deixe
`OIDC_AUDIENCE=authenticated`. A inicializacao descobre a configuracao OIDC e as
chaves JWKS do projeto; use uma chave de assinatura assimetrica (por exemplo,
ES256), pois chaves simetricas nao podem ser verificadas por uma chave publica.
`GET /health` retorna `200` quando o banco esta
disponivel e `503` quando a verificacao falha.

As migrations versionadas sao aplicadas com `go run ./cmd/migrate`. O comando
cria `schema_migrations`, aplica os arquivos SQL em ordem e ignora versoes ja
registradas. No Render, execute-o no Shell do servico depois de configurar as
variaveis `DB_*`.

## Autenticacao e autorizacao

Todas as rotas, exceto `GET /health`, exigem access token Supabase no header
Bearer. O backend valida assinatura, issuer, audience e expiracao via OIDC/JWKS
e associa o subject/e-mail do token a um usuario ativo no banco. Desabilite o
cadastro publico no Supabase e mantenha a confirmacao de e-mail habilitada;
provisione apenas contas verificadas que correspondam a usuarios ativos locais.
Para conceder `super_admin`, configure `app_metadata.roles` no Supabase e a
marca local de superadmin; ambas precisam estar presentes.

```bash
curl -H "Authorization: Bearer <access-token>" http://localhost:3333/institutions
```

Sem token valido a API retorna `401`; usuarios sem cadastro ativo recebem `403`.

## Eventos

O WebSocket `GET /events` exige token no subprotocolo `bearer.<access-token>` e transmite eventos com
`eventId`, `type`, `version`, `source`, `correlationId`, `occurredAt` e
`payload`. Eventos sao persistidos em `audit_events` antes da distribuicao.

## Desenvolvimento local

Inicie MariaDB, configure as variaveis `DB_*` e `OIDC_*`, aplique as
migrations e execute `go run ./cmd/server`. O guia do monorepo
`mfe-communication` documenta a inicializacao coordenada dos MFEs e desta API.
