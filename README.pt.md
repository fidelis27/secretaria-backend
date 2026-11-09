# Secretaria Backend

API Go da Secretaria Escolar: instituições, estudantes, matrículas, turmas e usuários, com autorização por escopo e fluxo de eventos de domínio em tempo real. Os microfrontends que consomem esta API ficam no repositório [`mfe-communication`](https://github.com/fidelis27/mfe-communication).

A versão em inglês está no [`README.md`](README.md).

## Stack

Go, MariaDB, Supabase Auth (OIDC/JWKS), WebSocket e GitHub Actions.

## Arquitetura

Arquitetura hexagonal, com as dependências apontando para dentro:

```text
cmd/                 server, migrate, seed (composition roots)
internal/
  domain/            entidades e regras: authorization, enrollment, event,
                     group, institution, student, user
  application/       casos de uso e barramento de eventos em processo
  adapters/
    httpserver/      handlers, middleware e WebSocket
    mariadb/         repositórios (por interfaces do domínio)
    oidc/            verificação de tokens
  platform/          configuração e catálogo de erros
migrations/          migrações SQL versionadas
seeds/dev/           dados exclusivos de desenvolvimento
docs/adr/            registros de decisões de arquitetura
```

Os repositórios implementam interfaces do domínio, `main` conecta as dependências e as regras de negócio (incluindo autorização) ficam na camada de domínio, não no SQL nem nos handlers. Consulte o [guia de arquitetura](docs/architecture.md).

## Configuração

O servidor lê variáveis de ambiente. `.env.example` contém exemplos para desenvolvimento local; substitua as senhas de banco claramente falsas no seu próprio ambiente.

| Variável | Padrão | Finalidade |
| --- | --- | --- |
| `PORT` | `3333` | Porta do servidor HTTP. |
| `APP_ENV` | `development` | Controla os padrões de produção e o comportamento local de CORS. |
| `DB_HOST`, `DB_NAME`, `DB_USER` | nenhum | Configuração de conexão com MariaDB. |
| `DB_PORT` | `3306` | Porta do MariaDB. |
| `DB_PASSWORD` | vazio | Senha do MariaDB. |
| `DB_TLS` | `false` | Habilita TLS para MariaDB. |
| `DB_TLS_SKIP_VERIFY` | `false` | Desabilita a verificação do certificado do banco; não habilite em produção. |
| `OIDC_ISSUER` | nenhum | URL do issuer Supabase Auth, incluindo `/auth/v1`. |
| `OIDC_AUDIENCE` | `authenticated` | Audience aceita para os tokens. |
| `AUTH_REQUIRE_VERIFIED_EMAIL` | `true` em produção, senão `false` | Exige a claim superior `email_verified: true`. |
| `RATE_LIMIT_PER_MINUTE` | `120` | Capacidade e taxa de reposição do bucket por usuário ou IP. |
| `CORS_ORIGINS` | vazio | Origins permitidas separadas por vírgula; configure explicitamente em produção. |
| `BOOTSTRAP_SUPER_ADMIN_EMAIL` | vazio | Conta promovida ou criada somente quando não existe superadmin local. |

## Autenticação

Todos os endpoints registrados, exceto `GET /health`, exigem um access token Supabase. Envie-o no header `Authorization: Bearer <token>` nas requisições HTTP; o WebSocket `GET /events` aceita o token no subprotocolo `bearer.<access-token>`. O verificador usa descoberta OIDC e JWKS, aceita somente `RS256` e `ES256`, valida issuer, audience e tempos do token com tolerância de relógio de 30 segundos, e limita requisições ao issuer/JWKS a uma por segundo. Em produção, a claim superior `email_verified` deve ser `true`; `user_metadata`, que pode ser editada, não é confiável. O token é associado a um usuário local ativo pelo `sub` estável (`users.auth_sub`). Tokens inválidos recebem `401`; usuários desconhecidos ou inativos recebem `403`.

## Autorização

Os papéis são definidos por escopo de instituição e turma. [A política de domínio](internal/domain/authorization/policy.go) define as regras, e os handlers HTTP as aplicam antes das operações. Um `super_admin` exige tanto a entrada em `app_metadata.roles` do Supabase quanto a flag local de superadmin. Administradores de turma não podem remover nem rebaixar o último administrador da turma; a API responde `409`. Listagens com escopo de instituição são filtradas pelas instituições visíveis ao solicitante. Consulte a [ADR 0003](docs/adr/0003-autorizacao.md).

## Rotas públicas

O único endpoint registrado da API que dispensa autenticação de identidade é `GET /health` (que também atende `HEAD` pelo mux HTTP do Go); ele retorna `200` quando a verificação de saúde tem sucesso e `503` caso contrário. Esta é a única operação OpenAPI com `security: []`.

O middleware CORS também responde `204` a requisições de preflight `OPTIONS` antes da autenticação. Essas respostas são preflight, não endpoints de dados; os headers CORS para navegadores só são enviados para origins permitidas. Todos os demais endpoints registrados exigem autenticação.

## Eventos de domínio

As escritas publicam eventos com `eventId`, `type`, `version`, `source`, `correlationId`, `occurredAt` e `payload`.

- Cada evento é salvo em `audit_events`, com seu escopo institucional, antes de ser transmitido. A persistência ocorre depois do commit da escrita da entidade, em uma transação separada; portanto, uma falha ao salvar o evento pode fazer com que ele se perca (consulte a [ADR 0001](docs/adr/0001-eventos.md)).
- `GET /events` é um WebSocket. O token vai no subprotocolo `bearer.<access-token>`, que é repetido no handshake. O servidor verifica a Origin do WebSocket, limita os frames a 1 KiB, usa ping/pong para detectar conexões inativas e encerra as conexões durante o desligamento gracioso.
- `GET /events/history` retorna eventos persistidos com escopo por instituição.

## Segurança

- **Verificação do token:** issuer OIDC, audience, assinatura e tempos do token são validados; somente `RS256` e `ES256` são aceitos. A verificação de e-mail é obrigatória por padrão em produção.
- **Isolamento entre instituições:** a autorização é aplicada pela política de domínio. A filtragem de eventos entre instituições é coberta por [`event_handler_test.go`](internal/adapters/httpserver/event_handler_test.go) (`TestEventVisibleToInstitutionsFiltersOtherScopes`) e [`event_history_handler_test.go`](internal/adapters/httpserver/event_history_handler_test.go) (`TestEventHistoryHandlerDoesNotLeakCrossInstitutionEvents`).
- **Limite de requisições:** um token bucket por usuário/IP se aplica a `POST`, `PATCH`, `DELETE` e `GET /events` (capacidade e reposição padrão de 120 requisições por minuto); requisições recusadas recebem `429` e `Retry-After`.
- **Headers de resposta:** todas as respostas recebem `X-Content-Type-Options: nosniff` e `Referrer-Policy: no-referrer`; respostas autenticadas recebem `Cache-Control: no-store`.
- **CORS:** em produção, as origins devem ser configuradas explicitamente em `CORS_ORIGINS`; fora de produção, também são permitidas origins de loopback localhost e 127.0.0.1. CORS é uma política de acesso do navegador, não autenticação. Conexões WebSocket são recusadas quando sua Origin não é permitida.

## Dados

- As migrações são aplicadas em ordem por `go run ./cmd/migrate` e registradas em `schema_migrations`.
- Transferências de matrícula encerram a matrícula de origem e abrem a de destino em uma única transação de banco, preservando o histórico.
- Seeds de desenvolvimento são separados das migrações e se recusam a executar quando `APP_ENV=production`.

## Execução local com Docker

A imagem Docker e o Compose são para desenvolvimento local. Em produção, o Render usa o build Go nativo.

1. Copie `.env.example` para `.env` e defina `OIDC_ISSUER` como a URL do issuer de um projeto Supabase (`https://<project-ref>.supabase.co/auth/v1`). As senhas de banco do exemplo são falsas; substitua-as no seu ambiente local.
2. Inicie o MariaDB e a API:

   ```bash
   docker compose up --build --detach api
   ```

3. Verifique a saúde da API e do banco:

   ```bash
   curl --fail http://localhost:3333/health
   ```

Dados de desenvolvimento opcionais podem ser carregados com `docker compose --profile seed run --rm seed`. O banco é persistido no volume nomeado `mariadb_data`. Origins de loopback são permitidas fora de produção, portanto não é necessário definir `CORS_ORIGINS` para frontends locais.

O smoke test de Docker exige Docker e `curl`:

```bash
sh scripts/smoke-compose.sh
```

## Como executar

```bash
go run ./cmd/migrate     # aplicar migrações
go run ./cmd/seed        # seeds de desenvolvimento (somente fora de produção)
go run ./cmd/server      # iniciar a API (porta padrão 3333)

go test ./...
go vet ./...
go test -race ./...
```

## Qualidade

Os testes cobrem domínio, aplicação, adaptadores HTTP, OIDC, configuração e migrações. A CI executa `go build ./...`, `go vet ./...`, `go test -race ./...` e o build da imagem Docker em pull requests.

## Documentação

| Documento | Conteúdo |
| --- | --- |
| [`docs/architecture.md`](docs/architecture.md) | Camadas, pipeline de requisições, identidade, autorização, fluxo de eventos, modelo de dados e deploy |
| [`docs/openapi.yaml`](docs/openapi.yaml) | Contrato da API HTTP e WebSocket |
| [`docs/adr/`](docs/adr) | Registros de decisões de arquitetura |
| [`docs/security-and-quality-review.md`](docs/security-and-quality-review.md) | Problemas de segurança e qualidade, com correções |
| [`README.md`](README.md) | Versão em inglês deste README |

## Decisões de arquitetura

| ADR | Decisão |
| --- | --- |
| [0001](docs/adr/0001-eventos.md) | Falhas de publicação de eventos não alteram uma escrita que já teve sucesso; um outbox transacional continua sendo a solução de longo prazo |
| [0002](docs/adr/0002-identidade.md) | Modelo de identidade |
| [0003](docs/adr/0003-autorizacao.md) | Autorização por escopo por meio de políticas de domínio |
| [0004](docs/adr/0004-local-docker-image.md) | Imagem Docker local multi-stage |

## Limitações conhecidas

- A entrega de eventos é best effort até existir um outbox transacional (ADR 0001).
- A autorização das rotas é aplicada em cada handler; ainda falta uma tabela declarativa de rotas com um teste que a percorra (ADR 0003).
- Ainda faltam hardening da imagem de produção, varredura automatizada de vulnerabilidades da imagem e execução do smoke test de Docker na CI.
- O OpenAPI é escrito manualmente; ainda faltam paginação, busca, idempotência, metas de cobertura, testes de integração MariaDB na CI e linters de segurança.
