# ADR 0004: imagem Docker local em Alpine

- Status: aceita
- Data: 2026-11-05

## Contexto

O backend precisa de um ambiente local reproduzivel com MariaDB, migrations e
API. O deploy de producao continua usando o build nativo do Render e nao deve
ser alterado por esta decisao.

## Decisao

Usar uma imagem multi-stage: compilar os binarios Go com `CGO_ENABLED=0` em
`golang:<versao>-alpine` e executa-los em `alpine:3.22`. A imagem runtime instala
certificados CA para chamadas OIDC/JWKS e executa como usuario nao-root. O
`wget` do BusyBox verifica `GET /health`, que tambem verifica a conexao com o
banco. A imagem inclui os comandos `server`, `migrate` e `seed`; o Compose usa
`seed` apenas sob perfil opcional.

## Alternativas consideradas

- Distroless: descartada porque nao inclui shell nem `wget`; implementar o
  healthcheck dentro de um binario exigiria codigo de aplicacao desnecessario.
- Uma unica imagem Go com ferramentas de build em runtime: descartada por
  aumentar a superficie da imagem.

## Consequencias

- A stack Docker destina-se somente ao desenvolvimento local; nao muda o
  `render.yaml` nem define a imagem de producao.
- A imagem e menor que o estagio de build, mas Alpine ainda contem utilitarios
  de runtime. A atualizacao da imagem base e a validacao de vulnerabilidades
  continuam sendo responsabilidades pendentes.
- O smoke test Compose precisa ser executado em um ambiente com Docker
  instalado antes de afirmar compatibilidade da stack.
