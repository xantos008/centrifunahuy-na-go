# Техническое задание: clean-room реализация Centrifugo OSS + функциональных эквивалентов Centrifugo PRO

**Дата исследования:** 15 ноября 2026  
**Название продукта:** Cedrazeka (DEC-12, 2026-11-15; бренд Centrifugo в продукте не используется)  
**Целевая реализация:** Go  
**Главное требование:** максимально совместимая по протоколу, API, конфигурации и поведению реализация публично документированных возможностей Centrifugo OSS и Centrifugo PRO.

> Это ТЗ описывает независимую clean-room реализацию по публичной документации и наблюдаемому публичному поведению. Не копировать закрытый код Centrifugo PRO, внутренние алгоритмы, тексты интерфейса, графику, товарные знаки или непубличные материалы. Не декомпилировать коммерческий бинарный файл и не обходить лицензионные ограничения. Совместимость протокола и реализация тех же общих функций — цель; копирование защищённой реализации — нет. Перед коммерческим выпуском провести юридическую проверку лицензий и совместимости.

> **Поправка от 2026-11-15 (решения DEC-01…DEC-09, см. §14).** Часть OSS строится на форке Centrifugo OSS v6.10.3 (Apache-2.0) с сохранением лицензий и attribution. Clean-room остаётся обязательным только для PRO. Где §0–§13 говорят о «независимой clean-room реализации» OSS, действует §14.

## 0. Цель, границы и определение «клона»

Создать самостоятельный высокопроизводительный realtime-сервер, который:
1. Принимает клиентские подключения и обеспечивает publish/subscribe по каналам.
2. Поддерживает совместимые клиентские протоколы, server API, JWT claims, ошибки, recovery semantics и конфигурационные сценарии.
3. Реализует все перечисленные ниже функции OSS.
4. Реализует функциональные аналоги всех публично перечисленных модулей PRO.
5. Работает в single-node и multi-node режимах, с горизонтальным масштабированием.
6. Имеет документацию, SDK compatibility tests, нагрузочные и fault-injection тесты, админ-панель, метрики и production deployment assets.

«Точный клон» трактовать как **проверяемую функциональную и протокольную совместимость с зафиксированной версией/срезом публичной документации**, а не как обещание идентичности недоступной внутренней реализации. Все неоднозначности оформлять в `docs/compatibility/known-differences.md`.

## 1. Источники истины и исследовательский процесс

Перед реализацией Claude Code обязан:
- Прочитать все Markdown-файлы репозитория: `CLAUDE.md`, `README*`, `docs/**/*.md`, `spec/**/*.md`, `adr/**/*.md`, `skills/**/*.md`, а также Markdown-файлы, найденные в подпроектах и подключённых пакетах, если они входят в область проекта.
- Построить инвентаризацию документации: путь, назначение, статус, противоречия, версия API.
- Изучить официальную документацию Centrifugo OSS и PRO, официальные release notes, публичные схемы API и официальные клиентские SDK.
- Для каждой функции зафиксировать URL источника, дату проверки, минимальную версию, конфигурацию, входы/выходы, ошибки, edge cases и тесты.
- Не выдумывать семантику. Если документация не отвечает на вопрос — добавить `OPEN QUESTION` и предложить эксперимент/тест совместимости.
- Сначала составить матрицу функций, затем архитектуру и ADR, и только потом писать код.

Основные публичные источники:
- https://centrifugal.dev/docs/getting-started/introduction
- https://centrifugal.dev/docs/getting-started/highlights
- https://centrifugal.dev/docs/server/
- https://centrifugal.dev/docs/pro/overview
- https://centrifugal.dev/pro
- https://github.com/centrifugal/centrifugo
- https://github.com/centrifugal/centrifugo-pro
- https://github.com/centrifugal/centrifugal.dev
- https://github.com/centrifugal/centrifuge-go
- https://github.com/centrifugal/centrifuge-js
- https://github.com/centrifugal/protocol

Версии OSS и PRO могут иметь разные номера. Зафиксировать выбранный compatibility baseline в `docs/compatibility/baseline.md`; не предполагать, что latest OSS и latest PRO соответствуют друг другу.

## 2. Полный перечень функциональных областей OSS

Ниже — обязательный scope. Каждая функция должна иметь спецификацию, конфигурацию, тесты и статус реализации.

### 2.1 Core messaging и протокол
- PUB/SUB, каналы, namespace, публикация в канал, fan-out всем подписчикам.
- Мультиплексирование многих channel subscriptions на одном клиентском соединении.
- Client protocol: JSON и Protobuf/binary режимы; совместимые command/reply/event envelopes, IDs, push frames, error codes.
- WebSocket bidirectional transport.
- SSE / EventSource.
- HTTP streaming.
- gRPC transport.
- WebTransport (если baseline поддерживает; отдельно отметить experimental status).
- WebSocket bidirectional emulation для клиентов/сред, где полноценный duplex транспорт недоступен.
- Batching и корректное упорядочивание сообщений.
- RPC с клиента к backend через realtime connection.
- Client-side и server-side subscriptions.
- Stream subscriptions, map subscriptions, shared-poll subscriptions, proxy subscriptions.
- Правила backpressure, write deadlines, ping/pong, heartbeat, idle timeout, graceful close.
- Ограничения размера кадра/сообщения, числа соединений, подписок, каналов и очередей.

### 2.2 Аутентификация и авторизация OSS
- JWT authentication для подключения и подписки.
- Проверка подписи, алгоритма, `exp`, `iat`, `nbf`, issuer/audience при наличии конфигурации.
- Connect proxy и subscribe proxy.
- Publish, RPC, refresh и другие поддерживаемые proxy hooks.
- Namespace-level channel permissions.
- Private channels и subscription token.
- Пользовательские данные соединения, `info`, `meta`, channel data.
- Origin/CORS и TLS termination/deployment guidance.
- Защита от повторного использования/подмены токенов в рамках возможностей публичного протокола.
- Единая модель ошибок авторизации и безопасные сообщения об ошибках.

### 2.3 History, recovery и presence
- Channel history: ограничение по размеру и TTL.
- Stream positions: epoch + offset (или эквивалент baseline).
- Recoverable subscriptions и восстановление пропущенных публикаций при reconnect.
- Cache recovery mode / получение последней публикации при подписке.
- Presence API и presence stats.
- Join/leave events.
- Семантика history/recovery при смене epoch, очистке истории, переполнении и потере данных.
- Раздельная конфигурация broker и presence manager.

### 2.4 Engines, brokers, storage и масштабирование OSS
- Memory engine для локальной разработки/single node.
- Redis engine и поддерживаемые Redis-compatible backends согласно baseline.
- Redis Cluster/sharding там, где это входит в OSS baseline.
- NATS broker, если он присутствует в выбранной OSS версии.
- PostgreSQL broker/controller/map broker там, где они входят в выбранный OSS baseline.
- PostgreSQL-only multi-node deployment, если поддерживается baseline.
- Разделение Broker и Presence Manager.
- Межузловой pub/sub, node discovery/coordination, распределённая presence и history.
- Консистентность, reconnect к backend, retry, bounded queues, circuit breaking.
- Очистка просроченных данных, schema migrations, readiness/liveness.
- Multi-tenant isolation и namespace-level limits.

### 2.5 Async consumers и интеграционные паттерны
- PostgreSQL transactional outbox/consumer.
- Kafka consumer.
- NATS JetStream consumer.
- Redis Streams consumer.
- Google Cloud Pub/Sub.
- AWS SQS.
- Azure Service Bus.
- Гарантии доставки, offset/ack, retry, poison messages, idempotency и ordering — строго по baseline и явно задокументировать.
- Возможность безопасно публиковать realtime-события из транзакционного потока данных.

### 2.6 Configuration, CLI и эксплуатация OSS
- JSON/YAML/TOML конфигурация.
- Environment variables и CLI flags с документированным precedence.
- `genconfig`, `checkconfig`, `gentoken` и совместимые полезные команды.
- Валидация конфигурации до запуска, понятные ошибки.
- TLS/автоматический сертификат, если baseline поддерживает.
- Admin UI для наблюдения и базового управления.
- Prometheus metrics; Grafana dashboard.
- Structured logs, log levels, graceful shutdown, health endpoints.
- Docker image, Helm chart, Kubernetes probes/resources, Linux packages.
- SBOM, reproducible builds, image signing и release checksums для нашего продукта.

## 3. Полный публично заявленный список Centrifugo PRO

Реализовать все следующие модули. Внутри каждого модуля сделать API/config/UI/metrics/storage/security/tests, где применимо.

### PRO-01. Push notifications API и lifecycle устройств
- Управление device tokens и device records.
- Регистрация, обновление, удаление, просмотр устройств.
- Провайдеры: Firebase Cloud Messaging (FCM), Apple Push Notification service (APNs), Huawei Messaging Service (HMS), Web Push/VAPID.
- Device topics и user topics; безопасное управление подписками на topics.
- Рассылка конкретным device IDs, topics и provider-native recipients/tokens/topics, если предусмотрено API.
- Асинхронные worker queues; Redis Streams и PostgreSQL queue backend.
- Delayed/scheduled push (`send_at`), отмена по уникальному идентификатору, retry и dead-letter strategy.
- Provider error handling, invalid-token cleanup, token health lifecycle.
- Timezone-aware delivery.
- Локализации.
- Notification templates.
- Per-user/device rate limiting.
- Provider-specific payload passthrough, не навязывать единый payload там, где публичный API передаёт нативный формат.
- Push status/interaction update API.
- ClickHouse analytics для push events.
- Возможность подключить внешний token storage, если это предусмотрено baseline.
- Строгая защита секретов provider credentials.

### PRO-02. Channel и user tracing
- Подписка оператора на поток протокольных кадров по channel или user ID.
- Просмотр сообщений/событий в реальном времени через Admin UI и API.
- Фильтры, временные рамки, ограничение объёма и sampling.
- Сохранение trace в файл для последующего анализа.
- Redaction чувствительных полей и токенов.
- RBAC/audit для доступа к трассировке.
- Ограничение влияния tracing на latency и memory; bounded buffers и drop counters.

### PRO-03. ClickHouse real-time analytics
- Экспорт событий publications, connections, subscriptions, operations и push notification events.
- Текущие показатели, trends, агрегаты, временные диапазоны.
- Drill-down до connection resolution, где поддерживается.
- Connection flight recorder / аналитическая история в Admin UI.
- Retention policies, TTL, partitioning, materialized aggregates.
- Batch ingestion, retry, backpressure, schema versioning.
- Дашборды и API для аналитики.
- Контроль PII, configurable field suppression и tenant isolation.

### PRO-04. Operation rate limits
- Rate limiting операций realtime API для защиты от misuse/buggy clients.
- Лимиты по operation type, connection/user/channel/namespace/client labels и глобально — в рамках конфигурационной модели.
- Token bucket или иной алгоритм, выбранный по измерениям; не заявлять идентичность внутреннему алгоритму без подтверждения.
- Cluster-aware лимиты, fairness, burst, expiry и метрики.
- Предсказуемый error response и retry guidance.
- Отдельно ограничить стоимость дорогих операций (history, presence, subscribe, publish, RPC, admin/API).

### PRO-05. Admin UI SSO через OIDC
- OIDC-compatible identity providers.
- Authorization Code + PKCE, state/nonce, issuer/audience/signature checks.
- Проверка полномочий на каждом запросе.
- Session lifecycle, logout, token expiry/refresh, CSRF protection.
- Mapping claims/groups/roles.
- Не хранить provider secrets в UI/client bundle.

### PRO-06. Admin roles и audit log
- Роли admin и viewer, назначаемые по identity claims.
- Server-side authorization на каждом admin endpoint.
- Audit log: логины и каждое изменение через Admin UI/Admin API.
- Кто, когда, IP, действие, target, результат; correlation/request ID.
- Защита от изменения/удаления аудита обычным администратором.
- Retention/export и фильтрация аудита.

### PRO-07. Connections и channels snapshots
- Snapshot текущего состояния connections и channels на всех узлах кластера.
- Инспекция: кто подключён, user ID, client info, labels, subscriptions, channel stats.
- Фильтрация, пагинация, экспорт.
- Consistent-enough snapshot semantics: timestamp, node coverage, partial-result indicators.
- Защита персональных данных и RBAC.
- Не смешивать snapshot с постоянно актуальным Connections API.

### PRO-08. Configuration viewer
- Просмотр effective/normalized runtime configuration в Admin UI.
- Секреты, токены, пароли, private keys всегда redacted.
- Указание источника значения (file/env/flag/default), если доступно.
- Разграничение доступа.

### PRO-09. Enhanced observability и metrics
- Дополнительные granular metrics.
- Namespace resolution dimensions.
- Client SDK name breakdown.
- Prometheus-to-OpenTelemetry metrics bridge.
- Per-node CPU/RSS/process stats в Admin UI.
- Системные показатели брокеров, очередей, consumers, push workers, ClickHouse exporters.
- Cardinality guardrails для labels; никогда не добавлять произвольные user IDs в Prometheus labels.
- Официальный dashboard и alert rules.

### PRO-10. JWT metadata extraction и multi-JWKS
- Извлечение metadata/labels из JWT claims.
- Несколько JWKS providers с раздельной конфигурацией.
- Key rotation, cache TTL, background refresh, stale-key policy, issuer/audience/algorithm constraints.
- Безопасное поведение при недоступности JWKS.
- Поддержка mapping claim → connection meta/labels по документированным правилам.

### PRO-11. Client labels
- Immutable `map[string]string` на время жизни соединения.
- Получение labels из JWT claim, mapping `labels_from_claim` и connect proxy response.
- Использование labels в Prometheus dimensions (с ограничениями cardinality), Connections API, CEL, analytics, snapshots и outgoing proxy requests.
- Labels нельзя менять клиентской командой после connect.
- Валидация ключей/значений и лимитов размера.

### PRO-12. Server API enhancements
- JWKS-based authentication для HTTP/gRPC server API.
- `label_filter` и `all_users` для targeting операций.
- Targeting по labels для disconnect, refresh, subscribe, unsubscribe и других документированных операций.
- Проверка полномочий и защита от случайного fan-out на всех пользователей.
- Pagination и ограничения размера выборок.

### PRO-13. User status API
- API обновления активности пользователя (`update_user_status`-подобная операция).
- Хранение last-active/status с TTL в Redis или выбранном storage.
- Batch query для списка users.
- Семантика online/active/last seen — задаётся конфигурацией и документируется.
- Не подменять presence: user status и connection presence — разные понятия.

### PRO-14. Connections API
- Поиск/просмотр активных sessions/connections по user ID или выражению/фильтру, без обязательного использования presence.
- Metadata на соединениях для server-side inspection/session management.
- Pagination, filters, labels, node, client, user.
- Возможность targeting по результатам с безопасными лимитами.

### PRO-15. User blocking и token revocation
- Block/unblock users.
- Revocation tokens по ID.
- Инвалидация пользовательских токенов по issue time/временной границе.
- Немедленное применение к текущим соединениям, где требуется, и к будущим connect/subscribe.
- Cluster-wide propagation, durable state, TTL/cleanup.
- Защита от обхода через refresh/reconnect.
- Audit trail и админские права.

### PRO-16. Additional event hooks
- Channel state events: `occupied` / `vacated` при изменении наличия подписчиков.
- Cache empty events для lazy state population при cache miss.
- Webhook/proxy delivery, retry, timeout, deduplication и защита от event storms.
- Документировать ordering и delivery guarantees.

### PRO-17. Channel capabilities
- Connection-wide capabilities из JWT `caps` claim или connect proxy.
- Subscription capabilities из subscription JWT `allow` или subscribe proxy.
- Операции: `sub`, `pub`, `prs`, `hst` (подтвердить baseline).
- Exact/wildcard/regex matching согласно публичной семантике.
- Capability grants объединяются по правилам документации; не трактовать более специфичное правило как deny, если публичная семантика — grant-only.
- Expiration, precedence относительно namespace permissions, error codes.
- Unit/property tests на overlap и конфликтные patterns.

### PRO-18. Channel patterns
- Route-like channel patterns, например `/users/:name`, `/events/:project/:type`.
- `:var` захватывает один сегмент.
- Static pattern precedence над variable patterns.
- Проверка конфликтов при старте.
- Expose captured vars в CEL/proxy/transforms.
- Опция включения; неизвестный pattern даёт документированную ошибку.
- Проверить ограничения символов, wildcard и Unicode по baseline.

### PRO-19. Channel CEL expressions
- CEL-based expressions для custom channel-operation permission gates.
- Context: user, connection, channel, vars, labels, meta, operation и только документированные поля.
- Ограничения CPU/allocations/time/AST size; compile-on-config, cache compiled programs.
- Fail-closed при ошибках.
- Отдельные expressions для connect/subscribe/publish и иных операций только если это подтверждено docs.
- Тесты sandboxing, deterministic evaluation и resource exhaustion.

### PRO-20. Performance/scalability enhancements
- Singleflight/deduplication для конкурентных одинаковых операций.
- Shared position synchronization.
- Устранение лишних backend round trips и распределённых fan-out затрат.
- Измеримые perf budgets и benchmark suite.
- Не переносить неописанные внутренние алгоритмы; реализовать эквивалентный результат.

### PRO-21. Per-namespace engines
- Возможность назначить разные broker/presence backends разным namespaces.
- Например, Redis для одного realtime workload и PostgreSQL/NATS для другого.
- Runtime engine registry, config validation, lifecycle и graceful shutdown.
- Проверка изоляции history/presence и совместимости node cluster.

### PRO-22. Custom Controller
- Независимая конфигурация controller backend для control-plane traffic.
- Redis и NATS controller backends согласно публичному описанию.
- Изоляция control traffic от channel/broker traffic.
- В OSS PostgreSQL controller уже может существовать; не маркировать его как эксклюзивный PRO.
- Cluster coordination, propagation, retry, observability.

### PRO-23. Bandwidth optimizations
- Delta compression для at-most-once delivery, совместимая с клиентским протоколом.
- Channel compaction.
- Publish debouncing.
- Контроль размера и CPU cost; корректная работа с recovery/history.
- Метрики экономии bandwidth и отключение по namespace/channel.

### PRO-24. Message batching control
- Настраиваемый размер/таймаут/режим batching при записи в client connections.
- Per-transport/per-namespace overrides, если поддерживаются.
- Низкая latency для интерактивных событий и throughput для массовой рассылки.
- Ограничение максимальной задержки и размера batch.

### PRO-25. Server-side publication tags filter
- Tags на публикациях.
- Фильтрация публикаций для отдельных подписчиков на сервере.
- Работает для stream и map subscriptions согласно публичной матрице.
- Безопасное поведение при отсутствующих/невалидных tags.
- Нельзя полагаться на клиентскую фильтрацию для security boundary.

### PRO-26. Map subscriptions enhancements
- In-memory cache layer.
- PostgreSQL read replicas.
- PostgreSQL broker fan-out.
- Redis Cluster support.
- Per-namespace map brokers.
- Cache invalidation, consistency model, stale reads, backpressure.
- Отдельные тесты на multi-node и failover.

### PRO-27. Shared poll enhancements
- Instant initial data через cached items.
- Delta compression.
- Notification fast path.
- Adaptive backpressure.
- Shared poll relay — **не считать обязательной GA-функцией**, так как в публичной матрице он обозначен как work in progress/in progress. Реализовывать только как отдельный экспериментальный модуль с feature flag, если появится подтверждённая спецификация.

### PRO-28. Additional granular observability
- Дополнительные insights по состоянию системы, брокеров, channels, connections, workers и transports.
- Correlation между traces, metrics и analytics без high-cardinality labels.
- Диагностические endpoints с RBAC.

## 4. Архитектурные требования

### 4.1 Язык и структура
- Go — основной язык серверной части.
- Разделение на `cmd/`, `internal/`, `pkg/`, `api/`, `web/`, `deploy/`, `docs/`, `tests/`.
- Чёткие интерфейсы для Transport, Protocol, Authenticator, Authorizer, Broker, Presence, History, MapStore, Controller, PushProvider, AnalyticsSink, Tracer.
- Не создавать монолитный пакет; не связывать transport напрямую с конкретной БД.
- Dependency injection через явную композицию; минимум глобального состояния.
- Context cancellation, bounded goroutines, graceful shutdown.
- Строгая типизация; не использовать `any` без обоснования и не маскировать ошибки type assertions.

### 4.2 Предлагаемые подсистемы
1. `protocol`: JSON/Protobuf codec, commands, replies, push frames, errors.
2. `transport`: WebSocket, SSE, HTTP-stream, gRPC, WebTransport (feature-gated).
3. `client`: connection lifecycle, subscriptions, write queues, batching, backpressure.
4. `auth`: JWT, JWKS, proxy auth, metadata extraction, revocation.
5. `authorization`: namespaces, capabilities, CEL, channel patterns, tags.
6. `pubsub`: broker abstraction, fanout, history, recovery, positions.
7. `presence`: presence manager + join/leave.
8. `map`: map subscriptions and map broker.
9. `sharedpoll`: poll subscriptions and notification path.
10. `controller`: cluster-wide control plane.
11. `serverapi`: HTTP/gRPC API and API authentication.
12. `consumers`: DB/queue CDC and outbox integrations.
13. `push`: device registry, topics, provider adapters, queues, templates/locales.
14. `analytics`: ClickHouse event pipeline and query API.
15. `tracing`: channel/user trace pipeline and file export.
16. `admin`: UI, OIDC, RBAC, snapshots, config viewer, audit.
17. `observability`: Prometheus, OTEL, logs, dashboards.
18. `config`: schema, loaders, env/flags, validation, hot reload only where safe.

### 4.3 Storage
- Redis: ephemeral state, presence, distributed coordination, queues and/or broker depending on adapter.
- PostgreSQL: durable device registry, optional broker/history/map/controller, audit metadata and migrations.
- ClickHouse: analytics/event history.
- NATS: broker/controller/queue adapter where required.
- Все storage adapters должны иметь contract tests и integration tests.
- Не обещать exactly-once delivery без формального доказательства; по умолчанию проектировать at-least-once + idempotency там, где это применимо.

### 4.4 API compatibility
- Сохранить имена команд/методов, request/response shapes, defaults, error codes и wire semantics по выбранному baseline.
- OpenAPI для HTTP API; protobuf schemas для gRPC/client protocol, если применимо.
- Golden fixtures, differential tests против официальных SDK и публичного reference server (только легально доступного).
- Compatibility suite должна тестировать JSON и binary protocol.
- Версионировать внутренние API и события аналитики.

### 4.5 Admin UI
- Отдельный frontend package.
- Экран connections, channels, namespaces, nodes, metrics, traces, snapshots, analytics, push devices/jobs, audit log, config viewer.
- OIDC/PKCE и RBAC проверяются backend-ом, не только UI.
- Viewer не может выполнять mutations.
- Все опасные действия требуют подтверждения и audit entry.

## 5. Security baseline — обязательные требования
- Threat model до начала реализации.
- TLS in transit; encryption at rest — через возможности storage/deployment.
- JWT algorithm allowlist; запрет `alg=none`; issuer/audience checks.
- JWKS SSRF protection: allowlisted URLs/domains, DNS/IP validation, redirect policy, bounded response size/timeouts.
- OIDC state/nonce/PKCE, CSRF, secure cookies, session rotation.
- Rate limits и quotas на connect/subscribe/publish/history/presence/RPC/admin/server API.
- Limits на payload, channel length, labels, metadata, subscriptions, queue depth и CEL execution.
- Secrets только через secret manager/env; не логировать credentials/tokens/payload PII.
- CEL sandbox/fail-closed.
- Audit log для privileged actions.
- Tenant isolation во всех слоях.
- Fuzzing протокола, JWT, parsers, config, channel patterns и CEL.
- Dependency scanning, SBOM, SAST, container scanning.
- Security review для push providers, webhook SSRF и external URLs.

## 6. Производительность и надёжность: измеримые цели

Не выдумывать throughput заранее. Сначала создать reproducible benchmark harness и записать baseline на фиксированном железе. Затем согласовать SLO. Минимальные сценарии:
- 1:1, 1:N, N:N fanout.
- 1k/10k/100k/1M concurrent connections (поэтапно).
- 1/10/100/1000 channels per connection.
- Small/medium/large payloads, JSON и Protobuf.
- Redis/NATS/PostgreSQL backends.
- Reconnect storms и recovery.
- Slow consumers, network partitions, broker outage, node restart.
- Push queue throughput, provider throttling, retry storm.
- ClickHouse unavailable/backpressure.
- CEL worst-case expressions.
Метрики: p50/p95/p99 end-to-end latency, throughput, CPU, RSS, allocations, GC pauses, network bytes, backend ops, dropped messages, queue depth, reconnect/recovery success.

## 7. Тестовая стратегия и Definition of Done

Для каждой функции:
- Unit tests.
- Contract tests.
- Integration tests с реальными Redis/PostgreSQL/NATS/ClickHouse в Testcontainers или docker compose.
- Multi-node tests.
- Failure/recovery tests.
- Security tests.
- Load/performance tests для hot paths.
- Документация config/API/limits.
- Metrics/logs и operational runbook.

Обязательные общие тесты:
- Protocol golden tests и SDK interoperability.
- Fuzzing parsers и command handlers.
- Auth matrix: valid/expired/revoked/wrong issuer/wrong audience/key rotation.
- Recovery: no loss/expected gap/epoch changed/history expired.
- Ordering semantics и duplicate delivery.
- Multi-node fanout и presence correctness.
- Graceful shutdown и rolling restart.
- Backpressure и bounded memory при медленном клиенте.
- OIDC security suite.
- Push invalid token cleanup, topic permissions, delayed jobs, retries.
- ClickHouse outage and replay.
- Snapshot partial-node behavior.
- Admin RBAC/audit enforcement.
- Rate limit distributed behavior.
- CEL fail-closed и resource exhaustion.
- Tenant isolation.

Функция считается DONE только когда тесты, docs, config validation, observability и security review готовы. «Код компилируется» не является Definition of Done.

## 8. План реализации — строго по вертикальным срезам

Ниже приведены группы требований исходного плана. Нумерация текущих этапов работ после решения о форке приведена в [матрице совместимости](docs/compatibility/feature-matrix.md#implementation-order).

### Phase 0 — Reconnaissance и compatibility contract
- Прочитать все `.md`.
- Зафиксировать baseline версии.
- Создать feature matrix: `feature`, `OSS/PRO`, `source`, `config`, `API`, `storage`, `tests`, `status`, `open questions`.
- Изучить протокол и SDK.
- Создать ADR: protocol, storage, broker, transport, auth, multi-tenancy.
- Выход: docs и архитектура, никакого преждевременного большого кода.

### Phase 1 — Core MVP
- Config/CLI, protocol JSON, WebSocket, client lifecycle, PUB/SUB, memory broker, server HTTP API, JWT, namespaces, basic Admin UI, Prometheus.
- SDK interoperability и golden tests.

### Phase 2 — Production OSS core
- Redis/NATS/PostgreSQL adapters согласно baseline.
- History/recovery, presence, join/leave, SSE/HTTP-stream/gRPC.
- Proxies, RPC, private subscriptions, async consumers.
- Docker/Helm/observability.

### Phase 3 — Advanced subscriptions and authorization
- Map/shared-poll/proxy subscriptions.
- Binary Protobuf, WebTransport/emulation.
- CEL, capabilities, channel patterns, labels/tags.
- Compatibility and fuzz tests.

### Phase 4 — PRO security and administration
- JWKS multi-provider, JWT metadata extraction, server API JWKS auth.
- Connections API, user status, block/revoke.
- OIDC/PKCE, admin roles, audit, snapshots, config viewer.
- Rate limits and enhanced metrics.

### Phase 5 — PRO observability and performance
- Tracing, ClickHouse analytics, flight recorder.
- Singleflight/shared position sync.
- Delta compression, compaction, debouncing, batching.
- Per-namespace engines and custom controller.
- Map/shared-poll enhancements.

### Phase 6 — Push platform
- Device storage/lifecycle/topics.
- FCM/APNs/HMS/WebPush adapters.
- Queue workers, scheduled delivery, templates/locales/timezones, rate limiting, status analytics.
- Mobile/web integration examples.

### Phase 7 — Hardening and release
- Full compatibility suite, load and chaos tests.
- Security audit, SBOM, signed artifacts.
- Upgrade/migration docs, Helm chart, dashboards, SLOs, backup/restore.
- Release candidate only after all mandatory acceptance criteria pass.

## 9. Required project artifacts
- `CLAUDE.md`
- `README.md`
- `docs/compatibility/baseline.md`
- `docs/compatibility/feature-matrix.md`
- `docs/compatibility/known-differences.md`
- `docs/architecture/overview.md`
- `docs/architecture/adr/`
- `docs/security/threat-model.md`
- `docs/api/`
- `docs/operations/`
- `docs/testing/`
- `docs/research/source-register.md`
- `docs/research/open-questions.md`
- `docs/research/pro-feature-inventory.md`
- `scripts/` for reproducible tests/benchmarks
- `deploy/docker/`, `deploy/helm/`
- `.github/workflows/` CI, security, release

## 10. AI tooling / Claude Code bootstrap

Claude Code must inspect installed tools and project docs before installing anything. Do not blindly install every item from the public skills/MCP ecosystem: a skill or MCP server is executable or instruction-bearing third-party code and may exfiltrate source code/secrets. Install the curated relevant set below, pin versions, inspect source/license, use project scope where possible, and record all installations in `docs/ai/tooling-lock.md`.

### Recommended Claude Code skills (create as project skills if no vetted maintained equivalent exists)
1. `go-best-practices` — Go idioms, errors, context, interfaces, concurrency.
2. `go-concurrency` — goroutines, channels, locks, race analysis, backpressure.
3. `go-performance` — pprof, benchmarks, allocation/GC, flamegraphs.
4. `go-testing` — table tests, integration, Testcontainers, fuzzing.
5. `distributed-systems` — consistency, ordering, retries, idempotency, partitions.
6. `realtime-messaging` — pub/sub, WebSocket lifecycle, presence, recovery, fanout.
7. `protocol-engineering` — wire compatibility, codecs, protobuf, golden fixtures.
8. `redis-engineering` — Redis Streams/PubSub/Cluster, Lua, key design.
9. `postgresql-engineering` — schema, indexes, transactions, LISTEN/NOTIFY, partitioning.
10. `nats-engineering` — JetStream, consumers, ack/replay.
11. `clickhouse-engineering` — event schemas, MergeTree, TTL, ingestion, query tuning.
12. `grpc-protobuf` — protobuf schemas, gRPC streaming, compatibility.
13. `websocket-engineering` — RFC 6455, heartbeat, proxy behavior, limits.
14. `security-review` — threat modeling, authz/authn, secrets, SSRF, CSRF.
15. `jwt-jwks-oidc` — JWT validation, JWKS rotation, OIDC PKCE.
16. `cel-policy-engine` — CEL compilation, sandboxing, fail-closed policy.
17. `observability-otel-prometheus` — metrics, traces, cardinality, dashboards.
18. `load-testing` — k6/vegeta/wrk scenarios, capacity methodology.
19. `chaos-engineering` — fault injection, resilience and recovery.
20. `frontend-admin-dashboard` — accessible admin UI architecture.
21. `react-typescript` — typed frontend implementation if React is selected.
22. `playwright-e2e` — UI/API end-to-end and regression testing.
23. `docker-kubernetes-helm` — containers, probes, Helm, rollout.
24. `api-documentation` — OpenAPI/protobuf docs and examples.
25. `dependency-supply-chain` — SBOM, provenance, signing, vulnerability scanning.
26. `technical-writing` — ADRs, specs, runbooks and compatibility matrix.
27. `code-review` — correctness, concurrency, security and maintainability review.
28. `migration-compatibility` — differential testing and version migration.

### MCP servers / integrations to evaluate
Install only if the server is maintained, has a clear license, supports least privilege, and is useful to the repository:
- `filesystem`: scoped access to project directory only.
- `github`: issues, PRs, code search and repository metadata; do not grant write access unless needed.
- `context7` (or equivalent documentation MCP): versioned official library documentation.
- `postgres`: local/dev database inspection only; never production credentials by default.
- `redis`: local/dev Redis inspection only.
- `clickhouse`: local/dev analytics schema/query inspection only.
- `docker`: inspect local containers/images and compose.
- `kubernetes`/`helm`: only for a dedicated dev cluster and read-only by default.
- `playwright`: browser-based Admin UI tests.
- `fetch`/HTTP documentation reader: allowlisted documentation domains only.
- `memory`/project knowledge MCP: optional; never store secrets or customer data.
- `sequential-thinking`/planning MCP: optional convenience, not a source of truth.

Avoid generic shell MCPs with unrestricted host access, unknown “all-in-one” MCP bundles, production database credentials, browser sessions containing personal accounts, and tools that upload repository contents to third parties.

### Bootstrap procedure
1. Run `claude --version`, inspect `claude mcp list`, inspect project `.claude/` and user-level settings without printing secrets.
2. Recursively enumerate and read every project Markdown file before changing files.
3. Check whether each listed skill/MCP already exists; do not duplicate.
4. For each missing skill, search trusted official sources/maintained registries, inspect its full instructions and scripts, verify license and recent maintenance, pin a version/commit, then install project-locally where supported.
5. For MCP servers, inspect package, publisher, permissions, environment variables and network destinations. Add only the approved subset. Never guess package names or execute install commands copied from untrusted README files.
6. If an install requires credentials, paid access, or elevated permissions, stop and ask the user. Never invent credentials.
7. Record exact versions, source URLs, commit hashes, permissions and rationale in `docs/ai/tooling-lock.md`.
8. Run a minimal smoke test for each tool; remove tools that fail verification.
9. Continue with Phase 0. Do not start feature implementation until the feature matrix and baseline are reviewed.

## 11. Open questions that must be resolved before claiming full compatibility
- Exact OSS and PRO baseline versions and their protocol versions.
- Which transports are GA vs experimental in the selected baseline.
- Exact gRPC/client protocol schemas and compatibility guarantees.
- Precise semantics of each server API command, error code and default.
- Supported Redis, NATS and PostgreSQL versions and feature parity.
- Exact list of proxy hooks and async consumer guarantees in baseline.
- CEL context variables, allowed functions and expression lifecycle.
- Rate-limit scopes, algorithm and distributed consistency.
- Snapshot consistency and pagination guarantees.
- Analytics event schema, retention defaults and delivery guarantees.
- Push provider feature parity and exact provider payload contracts.
- Whether PRO-only feature descriptions have changed since the source snapshot.
- Features marked in progress (notably shared poll relay): exclude from GA scope until officially documented as released.

## 12. Acceptance criteria for “feature-complete”
The project may be called “OSS + PRO functional equivalent” only when:
- Every row in `feature-matrix.md` is `implemented` or explicitly `not applicable` with evidence.
- All documented API and protocol compatibility tests pass.
- All supported official client SDK integration tests pass.
- All mandatory storage adapters and multi-node tests pass.
- Every PRO module PRO-01…PRO-28 has its own acceptance suite and docs.
- Security review has no unresolved critical/high findings.
- Load, chaos, soak, upgrade and restore tests meet agreed SLOs.
- Admin UI RBAC and audit coverage is complete.
- No unsupported claim of exact internal parity is made.

## 13. Phase 0: проверка полноты и дополнения (2026-11-15)

Источник: исследование Phase 0 по baseline OSS v6.10.3 / PRO v6.10.3 / protocol v0.23.1 / docs `69da714`. Полная детализация: `docs/compatibility/feature-matrix.md` (339 строк), `docs/research/*-research.md`, `docs/research/pro-feature-inventory.md`. Этот раздел **дополняет** разделы 2–3 и имеет приоритет при расхождениях с ними.

### 13.1 Уточнения статуса (upstream), обязательные для scope

- Experimental upstream: WebTransport и HTTP/3, WebSocket over HTTP/2 (RFC 8441), map subscriptions и shared poll (вся функциональность, с v6.8.0, поддержка только в centrifuge-js), proxy subscription streams, PostgreSQL broker/controller/map broker (PG ≥16, с v6.8.0), Redis IAM auth (с v6.10.0).
- PRO: channel state events — preview; snapshots, `meta_from_claim`, multi-JWKS, channel compaction — beta; flight recorder, `update_push_status`, node-grouped sharded PUB/SUB, `client.batch_periodic_events` — experimental.
- Удалено upstream в v6: SockJS, Tarantool. Не реализуем.
- §2.1 «gRPC transport»: в OSS есть только **однонаправленный** uni gRPC и gRPC server API. Двунаправленного gRPC-транспорта для клиентов нет (решение DEC-03).
- §2.4: NATS broker — at-most-once, без history/recovery/presence. PostgreSQL broker есть (experimental), presence manager на PostgreSQL отсутствует. PostgreSQL controller — OSS (experimental). Redis/NATS controller — PRO-22.
- Multi-tenancy (§2.4, §5): в Centrifugo OSS/PRO нет сущности tenant. Документированы только паттерны: dynamic JWKS с `{{tenant}}`, PostgreSQL `table_prefix`, префиксы ключей Redis/NATS. Полноценная изоляция — расширение проекта (ADR-0014, DEC-02).

### 13.2 OSS-функции, отсутствовавшие в §2 (добавляются в обязательный scope)

1. Протокол:
   - async `send`;
   - `message` push;
   - строгий JSON-декодинг (v6.10.0);
   - `Error.temporary`;
   - headers emulation;
   - server time в connect;
   - client name/version.
2. Транспорты:
   - uni WebSocket / uni SSE / uni HTTP-stream как отдельные транспорты;
   - `/emulation`;
   - `/connection/init`;
   - permessage-deflate и `decompressed_message_size_limit`;
   - H2C;
   - `cf_ws_frame_ping_pong`;
   - маппинг connect-ошибок в disconnect-коды и HTTP-статусы для uni.
3. Каналы:
   - user-limited `#`-каналы;
   - `channel_regex`;
   - `publication_data_format`;
   - автоматический персональный канал и `single_connection`;
   - positioning отдельно от recovery;
   - per-namespace `history_max_publication_limit` / `recovery_max_publication_limit` (v6.10.3);
   - `auto_cache_recover`;
   - `history_meta_ttl`.
4. Delta compression (fossil) и client-side publication tags filter (`tf`, FilterNode).
5. Server API полностью:
   - `publish`, `broadcast`, `subscribe`, `unsubscribe`, `disconnect`, `refresh`, `presence`, `presence_stats`, `history`, `history_remove`, `channels`, `info`, `batch`;
   - недокументированный `rpc` (CORE-OQ-01);
   - `map_publish/remove/read_state/read_stream/stats/clear`, `shared_poll_publish`;
   - transport error mode;
   - idempotency key и versioned publish;
   - Swagger UI.
6. Аутентификация:
   - JWKS и dynamic JWKS (OSS, v6.7.0);
   - ротация HMAC (v6.6.1);
   - Ed25519 PEM;
   - `user_id_claim`;
   - отдельная конфигурация subscription token;
   - полные наборы claims (`info`, `b64info`, `channels`, `subs`, `meta`, `expire_at`, `override`, `jti`, `client` в subscription token);
   - `nbf` проверяется (INFRA-CONFLICT-10);
   - insecure-режимы;
   - `user_id_http_header`.
7. Прокси:
   - sub_refresh;
   - subscribe stream (uni/bidi);
   - map publish/remove;
   - shared poll refresh;
   - именованные прокси;
   - `emulated_headers`;
   - `static_headers`/`static_metadata`;
   - `status_to_code_transforms`;
   - `include_connection_meta`;
   - `binary_encoding`.
8. RPC namespaces и встроенный `rpc.ping`.
9. Engines:
   - Redis Sentinel;
   - app-level sharding (jump hash);
   - Redis-compatible хранилища;
   - presence-опции Redis;
   - NATS raw mode и `allow_wildcards`;
   - SQL-поверхность PostgreSQL transactional publishing (публичный контракт);
   - map broker backends.
10. Consumers:
    - publication data mode;
    - повтор только для ошибок 100/111;
    - graceful drain (v6.10.3).
11. Конфигурация и CLI:
    - `.env`;
    - `CENTRIFUGO_VAR_*`;
    - SIGHUP reload (только секреты и опции каналов);
    - `--watch-config-file`;
    - `shutdown.timeout`;
    - `pid_file`;
    - internal port;
    - `handler_prefix`;
    - команды `version`, `defaultconfig`, `defaultenv`, `configdoc`, `gensubtoken`, `checktoken`, `checksubtoken`, `checkconfig --strict`.
12. Наблюдаемость и эксплуатация:
    - Graphite;
    - native histograms;
    - OpenTelemetry tracing server API;
    - pprof;
    - подписи Sigstore и SBOM (для наших артефактов);
    - SDK spec (состояния, backoff, `getState`) как контракт поведения сервера.

### 13.3 PRO-модули и подфункции, отсутствовавшие в §3

| ID | Функция | Статус upstream |
|---|---|---|
| PRO-29 | Ephemeral client publications (JSON Schema, tag extraction) | страница скрыта, релиз не подтверждён — **исключено** до решения DEC-06 |
| PRO-30 | Dictionary compression | draft/unreleased — **исключено** |
| PRO-31 | Grouped publications (`publication_grouping_disabled`, `publication_grouping_max_delay`) | GA, включено по умолчанию с 6.9.7 |
| PRO-32 | Redis replicas: `subscribe_on_replica`, `presence_read_from_replica` | GA |
| PRO-33 | Redis Cluster sharded PUB/SUB: partitions, precomputed tags, node-grouped | GA / experimental |
| PRO-34 | Runtime-опции: `process_commands_off_read_loop`, compression prepared message cache, `use_fast_client`, `batch_periodic_events` | GA / experimental |
| PRO-35 | Sentry | GA |
| PRO-36 | License/sandbox | **не применимо**, не воспроизводить |
| PRO-37 | Admin Inspector, `admin.instance_name`, расширения status page | GA |
| PRO-38 | OTel: GCP ADC, resource detectors, `service.instance.id` | GA |
| PRO-39 | Метки `accept_protocol`, client names, client labels | GA |
| PRO-40 | PostgreSQL stream broker replicas + fan-out (6.10.0); PG controller read replica (документация отсутствует) | GA / unknown |
| PRO-41 | `database.postgresql.pool_max_size`, метрики пула | GA |
| PRO-42 | Подписи релизов и SBOM | GA (для нашего продукта) |

Также:
- client RPC `update_user_status` и `user_status.disable_for_client`;
- `admin.api_key` для trace API;
- uni `subs` проверяются через caps/CEL (6.5.1);
- graceful shutdown PRO-подсистем (6.10.3);
- обязательная SSRF-защита Web Push.

### 13.4 Утверждения §3, не подтверждённые документацией

Реализовывать **только как явно помеченные расширения** (`known-differences.md`), без заявления совместимости:
- **PRO-01:** pluggable external token storage (есть только raw-token режим); dead-letter queue (есть retry до `expire_at`, 24h, и счётчики drop).
- **PRO-02:** sampling, временные окна, redaction, отдельные RBAC/audit для tracing, bounded buffers.
- **PRO-03:** partitioning/materialized aggregates, analytics API, PII suppression, tenant isolation.
- **PRO-04:** лимиты по labels/каналу, глобальный cluster-wide total, лимиты admin/API.
- **PRO-05:** state/nonce, CSRF, refresh, явный session lifecycle.
- **PRO-06:** correlation ID, защита от изменения, retention/export. Audit пишется только в process log, server API не аудируется.
- **PRO-07 / PRO-14:** pagination, export, partial-result индикаторы, поле `node`.
- **PRO-08:** источник значения (file/env/flag).
- **PRO-10:** поведение кеша client JWKS.
- **PRO-11:** лимиты размера labels.
- **PRO-12:** операции, кроме `subscribe`/`unsubscribe`/`disconnect`/`refresh`/`connections`.
- **PRO-16:** deduplication/ordering.
- **PRO-19:** connect-CEL и переменная `operation` **не существуют**.
- **PRO-23:** метрики экономии bandwidth.
- **PRO-24:** per-transport overrides.
- **PRO-28:** не отдельная функция upstream, покрывается PRO-09/37/39.

Подтверждено:
- HMS;
- per-device и per-user push rate limit (через `device.user` в key);
- сохранение trace в файл;
- PRO-27 relay = WIP.

### 13.5 Решения, требующие пользователя

DEC-01…DEC-09 в `docs/research/open-questions.md` §A: reuse кода OSS, multi-tenancy, gRPC-транспорт, брендированные имена (env/metrics), смешанные кластеры, скрытые PRO-страницы, admin API, MCP, Helm.

## 14. Решения пользователя и изменения ТЗ (2026-11-15)

### 14.1 Принятые решения

| ID | Решение | ADR |
|---|---|---|
| DEC-01 | Основа проекта — Centrifugo OSS v6.10.3 (Apache-2.0, коммит `4b6d86e`). OSS не переписываем с нуля. `centrifuge` (MIT) — немодифицированная зависимость. Закрытый код PRO не используем, PRO-функции пишем сами по публичной документации. Происхождение каждого заимствованного компонента фиксируется | 0002, 0029, `docs/legal/provenance.md` |
| DEC-02 | Deployment-per-tenant в v1. Новые PRO-хранилища получают колонку `scope` (EXT-05), чтобы не блокировать будущую полноценную multi-tenancy | 0014 |
| DEC-03 | Двунаправленный gRPC-транспорт для клиентов не делаем. Остаются uni gRPC, gRPC server API и gRPC proxy | 0021 |
| DEC-04 | Свои имена по умолчанию, опциональный режим `compat: centrifugo` для имён конфигурации и API (env, метрики, имена gRPC-сервисов, заголовки Kafka). Юридические ограничения описаны в `docs/legal/naming-and-trademarks.md` | 0022 |
| DEC-05 | Смешанные кластеры с нодами Centrifugo не поддерживаются | 0023 |
| DEC-06 | Публично не документированные PRO-функции (PRO-29, PRO-30, PRO-27.05) не входят в обязательный scope. Неизвестное поведение фиксируем как open question | 0024 |
| DEC-07 | Admin API — только по подтверждённым публичным контрактам. Расширения живут под `/admin/x/v1/*`, помечены EXTENSION и не выдаются за совместимые | 0025 |
| DEC-08 | GitHub MCP и Context7 не ставим. Все инструменты остаются в проекте | 0026 |
| DEC-09 | Helm откладываем, пока Core не стабилен (конец этапа 3) | 0027 |

Дополнительно решено:
- **DEC-10** — лицензия Apache-2.0 на весь репозиторий: бесплатно для всех, «AS IS», без гарантий и без ответственности авторов (ADR-0030);
- **DEC-11** — формулировка «Mission» в CLAUDE.md обновлена;
- **DEC-12** — продукт называется **Cedrazeka** (`cedrazeka`, `CEDRAZEKA_`).

### 14.2 Категории функций

Каждая строка `docs/compatibility/feature-matrix.md` относится к одной категории:
- **UPSTREAM** — подтверждённая совместимость с Centrifugo OSS v6.10.3 (196 строк). Код унаследован от форка. Строка получает статус `verified` только после прохождения compatibility suite.
- **PRO** — подтверждённая функциональность Centrifugo PRO v6.10.3 (143 строки), реализуется по документации.
- **EXTENSION** — наши собственные возможности (EXT-01…EXT-17). Они выключены по умолчанию и не меняют UPSTREAM-поведение. Сюда же переходят утверждения §3, не подтверждённые документацией (§13.4).

### 14.3 Обязательный compatibility test suite (дополняет §7)

Одинаковые сценарии запускаются против `centrifugo/centrifugo:v6.10.3` (образ закреплён по digest) и против нашей сборки. Сравниваются нормализованные транскрипты:
- протокол (JSON и Protobuf);
- ответы и ошибки;
- авторизация;
- подписки;
- history, recovery, presence;
- server API (HTTP и gRPC).

Расхождение роняет CI, если оно не связано с принятой записью в `known-differences.md`. Спецификация: `docs/testing/compatibility-suite.md`, ADR-0028.

### 14.4 Лицензионные обязательства

- Не удалять и не переписывать лицензионные уведомления заимствованного кода.
- Изменённые Apache-2.0 файлы помечаются уведомлением об изменении.
- `NOTICE`, `THIRD_PARTY_LICENSES` и `upstream/BASE` создаются при импорте.
- Встроенный брендированный web UI upstream удаляется из сборки.

Реестр: `docs/legal/provenance.md`.

### 14.5 Изменения плана (заменяет §8 в части OSS)

- Этап 1 — импорт OSS, лицензии, переименование модуля, свои имена и режим совместимости, каркас compatibility suite. Детальный план: `docs/plan/phase-1.md`.
- Этапы 2–3 — подтверждение унаследованных engines, транспортов, прокси, consumers и API через compatibility suite (переписывание не требуется).
- Этапы 4–9 — PRO-модули, push, hardening; Helm — на этапе 9.

Таблица этапов: `docs/compatibility/feature-matrix.md#implementation-order`.

