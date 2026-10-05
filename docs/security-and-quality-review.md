# Security and quality review

| Problema encontrado | Risco | Correcao | Commit | Teste que cobre |
|---|---|---|---|---|
| Escrita confirmada seguida por falha de persistencia/publicacao do evento retornava 500; a criacao de matricula tambem nao emitia evento. | O cliente podia repetir uma operacao ja gravada; sem idempotencia, havia risco de duplicacao, e eventos de criacao de matricula faltavam. | A falha de evento passou a ser logada e nao muda a resposta de sucesso; criacao de matricula publica `ENROLLMENT_CREATED`. Esta e uma alternativa temporaria, nao um outbox. | `39af1cc`, `feat(events): publish enrollment creation` | `TestPublishDomainEventBestEffortLogsFailure`, `TestPublishDomainEventPersistsEnrollmentCreation` |
| O upgrade WebSocket aceitava o subprotocolo Bearer sem ecoa-lo no handshake e permitia Origins sem cabecalho. | Navegadores recusavam o handshake; clientes nao autorizados podiam tentar abrir o stream. | Ecoa o subprotocolo, exige Origin permitido, limita frames, usa ping/pong e encerra as conexoes no shutdown. | `e931405`, `a2fc873` | `TestEventHandlerNegotiatesBearerSubprotocol`, `TestEventHandlerRejectsDisallowedOrigin`, `TestEventConnectionsCloseRegisteredConnectionsOnShutdown` |
| A autenticacao nao tinha uma politica configuravel de verificacao de e-mail. | Um token sem verificacao podia autenticar em producao ou metadata editavel podia ser confundida com claim confiavel. | Le somente o claim de topo; exige `true` por padrao em producao e valida explicitamente o comportamento quando ausente. | `e931405` | `TestIdentityMiddlewareEmailVerificationPolicy`, `TestLoadAuthRequireVerifiedEmailDefaults` |
| Token OIDC usava algoritmos indicados pelo discovery sem uma allowlist local, sem leeway curta definida pelo servico e sem limitar requisicoes ao issuer/JWKS. | Algoritmos inesperados, diferencas pequenas de relogio e excesso de refresh de chaves podiam enfraquecer ou indisponibilizar autenticacao. | Permite apenas RS256/ES256, aplica leeway de 30 segundos e limita requisicoes OIDC/JWKS a uma por segundo por processo. | `e931405` | `TestVerifierRejectsWrongAudience`, `TestVerifierRejectsExpiredAndUnsupportedAlgorithmTokens`, `TestPacedRoundTripperLimitsUnknownKeyRefreshes` |
| Escritas e streams nao tinham limite basico de requisicoes, e respostas careciam de cabecalhos de protecao. | Abuso de recursos e cache acidental de respostas autenticadas. | Token bucket configuravel por usuario/IP, `Retry-After`, `no-store`, `nosniff` e `Referrer-Policy`. | `a2fc873` | `TestRateLimitMiddlewareReturnsRetryAfter`, `TestRateLimiterIsSafeForConcurrentRequests`, `TestSecurityHeadersMiddlewareAddsBrowserHeaders` |
| Logs HTTP podiam evoluir para registrar headers ou query strings sensiveis. | Exposicao de bearer token ou e-mail em logs. | Observabilidade registra somente metodo, caminho sem query, status, duracao e correlation ID. | `a2fc873` | `TestObservabilityLogsDoNotContainAuthorizationOrFullEmail` |
| Migration `007` contem um e-mail pessoal e ja faz parte do historico publico. | Reescrever migration aplicada quebra consistencia entre ambientes; novos e-mails literais ampliariam exposicao. | Migration `007` e legada e nao foi reescrita; bootstrap atual usa `BOOTSTRAP_SUPER_ADMIN_EMAIL`; teste impede e-mails literais em migrations fora da lista historica. | `test(migrations): block email literals` | `TestNewMigrationsDoNotContainEmailLiterals` |

## Licoes dos bugs de evento

O primeiro bug de escrita/evento passou porque os testes cobriam falhas de validacao e do banco antes da escrita, mas nao a falha posterior ao commit da entidade durante a publicacao do evento. O teste novo injeta uma falha no repositorio de eventos e verifica que a falha e reportada como nao fatal.

O WebSocket nao tinha um teste que inspecionasse o subprotocolo negociado: o teste anterior conectava sem solicitar um protocolo, comportamento aceito por clientes nao-browser mas rejeitado pelo navegador quando `Sec-WebSocket-Protocol` era enviado. O teste de handshake agora solicita o subprotocolo Bearer e verifica o valor negociado.

## Decisao sobre migration legada

A migration `007_promote_owner_super_admin.sql` e historica e contem o e-mail pessoal que foi usado naquele ambiente. Ela nao deve ser alterada; novas instalacoes usam `BOOTSTRAP_SUPER_ADMIN_EMAIL`, e o teste de migrations bloqueia novos enderecos literais.

## Pendencias desta revisao

O outbox transacional continua pendente. Tambem permanecem pendentes a reorganizacao declarativa de rotas, OpenAPI e padronizacao total dos erros, paginacao/busca/idempotencia, thresholds de cobertura, integracao MariaDB no CI, linters de seguranca, e imagem Docker endurecida.
