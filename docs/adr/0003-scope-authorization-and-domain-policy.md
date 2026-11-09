# ADR 0003: Autorizacao por escopo e politica de dominio

## Status

Aceita para o modelo atual; centralizacao declarativa das rotas ainda pendente.

## Decisao

O middleware autentica o usuario e disponibiliza a identidade no contexto. As politicas de dominio determinam superadministracao, escopo institucional e gestao de grupos; handlers aplicam essas politicas antes de executar operacoes. As rotas de API permanecem protegidas por autenticacao, exceto `GET /health`.

## Consequencias

- Decisoes de autorizacao dependentes de instituicao/grupo continuam usando o dominio em vez de duplicar regras SQL nos handlers.
- A autorizacao declarativa por rota e o teste que percorre a tabela de rotas ainda precisam ser implementados para reduzir a dispersao atual.
