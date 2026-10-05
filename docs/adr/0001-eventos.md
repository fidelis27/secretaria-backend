# ADR 0001: Publicacao de eventos apos escritas

## Status

Aceita como alternativa temporaria ao outbox transacional.

## Decisao

Nas rotas de criacao de estudante, criacao de matricula e transferencia, suspensao e reabertura de matricula, uma falha ao persistir/publicar o evento e registrada no log, mas nao altera a resposta de sucesso da escrita concluida.

## Contexto

Hoje as entidades e os eventos sao persistidos em operacoes separadas. A implementacao de outbox exige repositorios transacionais que coordenem entidades, eventos e escopos, alem de um dispatcher com ciclo de vida ligado ao servidor; essa mudanca atravessa as interfaces atuais de dominio e persistencia.

## Consequencias

- Uma falha de publicacao nao induz o cliente a repetir uma escrita que o backend ja confirmou.
- O comportamento e melhor que responder `5xx` depois de uma escrita persistida, mas nao garante entrega eventual.
- Se a persistencia do evento falhar, o evento pode ser perdido.
- Se o cliente nao receber a resposta apesar de o servidor tê-la enviado, uma repeticao ainda pode duplicar a escrita; esta alternativa nao implementa idempotencia.
- Um outbox transacional continua sendo a solucao necessaria para garantir entrega at-least-once.
