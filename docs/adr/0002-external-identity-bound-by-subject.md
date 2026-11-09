# ADR 0002: Identidade externa vinculada pelo subject

## Status

Aceita.

## Decisao

`users.auth_sub` e o identificador externo principal. O backend busca primeiro por `sub`; e-mail so e usado na primeira entrada para localizar um cadastro legado sem `auth_sub`, e nesse caso o subject e vinculado permanentemente. Se o subject ja estiver vinculado a outra conta, ou se o cadastro local tiver subject diferente, o acesso e negado.

## Consequencias

- Alteracoes de e-mail no provedor nao mudam a identidade local depois do primeiro vinculo.
- E-mail sozinho nao pode reassociar uma conta ja vinculada.
- Administradores devem provisionar usuarios locais antes do primeiro login.
