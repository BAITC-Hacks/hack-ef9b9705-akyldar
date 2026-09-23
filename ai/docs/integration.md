# Подключение AI к Go backend

## Пакет и ответственность

AI поставляется отдельным Go module `hackalem/ai`, Go 1.22+, стандартная библиотека. `cmd/demo` — локальный сервер с двумя endpoints. Ветка AI основана на `main` (`051f110`) и не включает backend/frontend. Ниже учтён код [ветки backend на `ba61fef`](https://github.com/BAITC-Hacks/hack-ef9b9705-akyldar/tree/ba61fef/backend), просмотренный при передаче. Это предложение подключения, а не выполненное объединение веток или проверка совместного приложения. `backend/go.mod` объявляет `module backend`, Go 1.25.0; для совместной сборки нужен Go 1.25 или новее. Не заменяйте его файлом `ai/go.mod`.

Основные API пакета:

```go
func DefaultConfig() Config
func LoadConfigFromEnv() (Config, error)
func NewService(cfg Config, provider Provider) (*Service, error)
func RegisterRoutes(mux *http.ServeMux, service *Service)
func NewHandler(service *Service) http.Handler
```

`Config` содержит `APIKey`, `Model`, `Timeout`, `MaxAttempts`, `ForceFallback`. `nil` в `NewService` выбирает OpenAI provider. Fake provider применяется в offline-тестах. Backend сохраняет ответственность за авторизацию, сохранение, rating, подтверждение человеком, публикацию и предложения команд.

Для локального Windows-стенда есть `ai/run.ps1`: из корня проекта выполните `powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\ai\run.ps1`. Параметр execution policy относится только к этому процессу. Launcher загружает разрешённые настройки из соседнего `.env` в окружение процесса и восстанавливает прежние значения после завершения. `.env` исключён из Git; `.env.example` не содержит секретов. Прямой запуск Go и подключение к backend по-прежнему читают только process env. Параметр `-ForceFallback` включает резервный режим, `-Address` меняет адрес сервера.

## Подключение к ServeMux

Внутри текущего модуля `hackalem/ai` минимальное подключение выглядит так:

```go
package main

import (
    "log"
    "net/http"
    "time"

    ai "hackalem/ai"
)

func main() {
    cfg, err := ai.LoadConfigFromEnv()
    if err != nil {
        log.Fatal("invalid AI configuration")
    }
    service, err := ai.NewService(cfg, nil)
    if err != nil {
        log.Fatal("cannot initialize AI service")
    }
    mux := http.NewServeMux()
    ai.RegisterRoutes(mux, service)
    server := &http.Server{
        Addr:              "127.0.0.1:8080",
        Handler:           mux,
        ReadHeaderTimeout: 5 * time.Second,
    }
    log.Fatal(server.ListenAndServe())
}
```

В существующем backend используйте его config, server и router; новый сервер создавать не нужно. `NewHandler(service)` также предоставляет `http.Handler` для адаптера router. Сохраняйте путь `/api/ai/questions` или `/api/ai/card` при передаче запроса. Передавайте исходный request context, чтобы отключение клиента отменяло внешний запрос и ожидание повторной попытки.

### Предлагаемое подключение к `origin/backend`

После объединения веток расположение каталогов должно быть `backend/` и `ai/` рядом. Сохраните оба модуля. В `backend/go.mod` добавьте локальную зависимость (эти изменения в AI-ветке не выполнены):

```go
require hackalem/ai v0.0.0

replace hackalem/ai => ../ai
```

`hackalem/ai` здесь локальное имя, не адрес опубликованного Go-пакета; `replace` связывает его с соседней папкой. В `backend/cmd/server/main.go` добавьте import `ai "hackalem/ai"`. После создания трёх repository и перед созданием существующего `http.Server` используйте композицию handlers:

```go
aiConfig, err := ai.LoadConfigFromEnv()
if err != nil {
    log.Fatal("invalid AI configuration")
}
aiService, err := ai.NewService(aiConfig, nil)
if err != nil {
    log.Fatal("cannot initialize AI service")
}
combined := http.NewServeMux()
ai.RegisterRoutes(combined, aiService)
combined.Handle("/", httpapi.NewRouter(taskRepository, teamRepository, proposalRepository))
```

В существующем `http.Server` задайте `Handler: httpapi.WithCORS(combined, frontendOrigin)` и `WriteTimeout: aiConfig.Timeout + 15*time.Second`. Остальные настройки backend и lifecycle сохраняются. Сейчас backend использует `WriteTimeout: 10*time.Second`, что короче стандартного AI deadline 15 секунд. Композиция оставляет сигнатуру `NewRouter` и его task/proposal routes прежними, передаёт URL и context без `StripPrefix`. Альтернатива — принять `*ai.Service` в `NewRouter` и вызвать `ai.RegisterRoutes` на его `mux`, обновив все callers и тесты backend.

Настройки AI передаются окружению процесса **backend**. `ai/run.ps1` запускает только отдельный demo и не загружает env в backend. Backend и demo по умолчанию используют порт 8080; для параллельной проверки задайте demo `-Address 127.0.0.1:8081`. Общий сервер запускает backend; второй AI-сервер ему не нужен.

В `backend/internal/httpapi/cors.go` внутри разрешённого origin добавьте `w.Header().Set("Access-Control-Expose-Headers", "X-AI-Mode")`. Существующий `WithCORS` уже обслуживает OPTIONS и `Content-Type`; default origin — `http://localhost:5173`. Разместите его снаружи объединённого handler и сохраните согласованное `FRONTEND_ORIGIN`. До изменения браузер не сможет прочитать mode при cross-origin запросе, хотя HTTP header будет в ответе.

Проверки запускаются отдельно из `ai/` и `backend/`: `go test ./...`, `go vet ./...`. Команда из корня и тесты одного module не проверяют соседний вложенный module. Приведённое подключение ещё не применено и требует отдельной проверки AI routes, старых backend routes, CORS и полного UI flow.

### Сохранение карточки и совместимость моделей

`backend/internal/model/task.go` содержит все 11 полей AI, а также `id`, `initial_description`, `rating`, `readiness_level`, `confirmed`, `published` и timestamps. Они остаются под управлением backend; AI JSON не расширяется. В просмотренном handler создание `POST /api/tasks` принимает `{initial_description, topic}`, а обновление `PUT /api/tasks/{id}` — 11 строк карточки. Поэтому после ручной проверки создайте draft с исходным `description` в `initial_description`, затем отправьте проверенную карточку в update endpoint. При повторном редактировании используйте существующий task ID. Массив `answers` в модели Task отсутствует: его хранение рядом с формой или отдельное расширение backend нужно согласовать, особенно при fallback.

Ошибки task/proposal API имеют форму `{ "error": "..." }`, ошибки AI — `{ "error": { "code": "...", "message": "..." } }`. Frontend должен обрабатывать обе формы либо команда добавит явный adapter. Эта поставка не меняет существующий backend error contract.

Fixtures — примеры для адаптации, не готовый импорт в API. Поля team (`name`, `interests`, `skills`, `technologies`) совпадают с просмотренной моделью. В proposals используйте реальные task/team IDs и входные `idea`, `plan`, `deadline`, `prototype_url`; fixture IDs, status и timestamps не подменяют значения, назначаемые backend. Оболочка `{synthetic, items}` и `source` карточки не являются body task/proposal endpoint.

## HTTP-контракт

| Путь | Запрос | Успешный body |
|---|---|---|
| `POST /api/ai/questions` | `{"description":"..."}` | `{"questions":["...","...","..."]}` |
| `POST /api/ai/card` | `{"description":"...","answers":[{"question":"...","answer":"..."}]}` | Объект с 11 строковыми полями карточки |

`description` обязателен и после trim не пуст. Для card `answers` обязателен, `[]` допустим. В каждом answer обязательны оба ключа; `question` непустой, `answer` может быть пустой. Нельзя передавать `null`, неизвестные ключи, неверные типы, повторяющиеся JSON keys, несколько объектов или текст вокруг JSON.

Card содержит ровно: `title`, `context`, `need`, `users`, `data`, `constraints`, `expected_result`, `success_criteria`, `contact`, `interaction_format`, `topic`. Все значения — строки, неизвестные сведения — `""`. В задании названы «12 полей», но перечислено 11; дополнительное поле требует отдельного согласования.

Схемы — единственный контракт формы ответа: [questions](../schemas/questions.schema.json), [card](../schemas/card.schema.json). Локальная проверка дополнительно исключает пустые и повторные вопросы. Допускается одна полная внешняя Markdown code fence в ответе модели; объект из постороннего текста не извлекается, повреждённый JSON не ремонтируется.

| Ограничение | Максимум |
|---|---:|
| HTTP request body | 64 KiB |
| `description` | 8 000 Unicode characters (Go runes) |
| Ответов пользователя | 20 |
| Один `question` во входе | 500 runes |
| Один `answer` во входе | 2 000 runes |
| Суммарный текст `question` + `answer` во всех answers | 16 000 runes |
| Один вопрос в ответе AI | 500 runes |
| Одно поле карточки | 8 000 runes |
| Внешний HTTP response body | 256 KiB |

Лимиты runes и bytes различаются: например, кириллица занимает несколько bytes в UTF-8. Любое ограничение размера входа возвращает 413; неверная структура/содержимое — 400.

## Статусы и заголовки

| Статус | Значение |
|---|---|
| `200` | Валидный live-результат или явно помеченный fallback |
| `400` | Ошибка JSON, обязательных полей, типов или непустоты |
| `413` | Превышен входной лимит |
| `415` | Неподдерживаемый Content-Type; нужен `application/json` |
| `405` | Метод не POST; ответ содержит `Allow: POST` |
| `408` | Родительский request context отменён или его deadline истёк; при закрытом соединении клиент может не получить ответ |

Ошибка имеет компактную форму `{"error":{"code":"...","message":"..."}}`. Код и английское сообщение безопасны для клиента: нет ключа, полного пользовательского ввода или сырого ответа внешнего API. UI должен обрабатывать статус/code и может локализовать текст самостоятельно. Ошибка пользовательского ввода никогда не маскируется успешным fallback.

Успех: `Content-Type: application/json`, `Cache-Control: no-store`, `X-AI-Mode: live | fallback`. Mode не добавляется в JSON. Frontend показывает рядом с карточкой явную подпись о резервном режиме и возможность ручного заполнения.

В AI-пакете встроенного CORS middleware нет. Если frontend находится на другом origin, host должен отвечать на preflight OPTIONS и разрешать конкретный origin, например текущий default backend `http://localhost:5173`:

```http
Access-Control-Allow-Origin: http://localhost:5173
Access-Control-Allow-Methods: POST, OPTIONS
Access-Control-Allow-Headers: Content-Type
Access-Control-Expose-Headers: X-AI-Mode
Vary: Origin
```

`Allow-Origin` выбирается из согласованного allowlist; `Expose-Headers` нужен на фактическом ответе, чтобы JavaScript мог прочитать режим. OPTIONS обрабатывается middleware до AI handler, иначе handler вернёт 405. Для frontend с тем же origin CORS не требуется. Если backend использует дополнительные auth headers/credentials, согласуйте их с его существующей CORS-политикой.

## Provider, повторы и fallback

OpenAI вызывается сервером через Responses API. В request: `model` из конфигурации, отдельные system prompt и JSON user data, `text.format` с `type: "json_schema"`, `name`, `schema`, `strict: true`. Актуальный синтаксис проверен по [официальному руководству](https://developers.openai.com/api/docs/guides/structured-outputs); [gpt-4.1-mini](https://developers.openai.com/api/docs/models/gpt-4.1-mini) — рекомендуемое начальное значение для коротких запросов. Доступность API проверяется отдельно от подписки в пользовательском интерфейсе.

На пользовательский запрос — максимум две попытки провайдера и общий timeout по умолчанию 15 секунд. Нет дополнительного слоя SDK retry. Временные connection errors, 429, 5xx и непригодный ответ могут использовать оставшуюся попытку с короткой задержкой в рамках общего deadline. Повтор не запускается для 401/403 или после отмены context. Пустой, отказной, обрезанный ответ, неправильный JSON/схема и обнаруженные guard неподтверждённые числовые/контактные/технологические сведения не выдаются пользователю как успешный live-результат. Другие смысловые ошибки guard может пропустить.

Отсутствующий ключ/модель и forced fallback не вызывают API. Внутренний AI timeout даёт fallback; отмена родительского request context прекращает работу. Body внешнего ответа ограничен и закрывается. Обычные логи не содержат API key или полного пользовательского текста.

Fallback questions — локальные RU/KK вопросы с эвристическим определением уже заполненных категорий. Эвристика ограничена: не распознаёт все перефразирования и может неточно определять язык. При полностью исчерпывающем описании минимум 3 вопроса и отсутствие повторов могут конфликтовать; модель и fallback не доказывают, что детали действительно отсутствуют.

Fallback card сохраняет исходный `description` дословно в `context`, остальные 10 полей пустые. Пользовательские ответы при fallback не превращаются автоматически в поля: это ручной черновик с доступным исходным контекстом. Frontend должен сохранять ответы рядом со своей формой до завершения ручной проверки.

## Проверка достоверности

Schema и строгий JSON parser проверяют только форму. Prompt разрешает исправлять грамматику, перефразировать без изменения смысла и объединять явные сведения. Нельзя добавлять названия, числа, данные, технологии, сроки, бюджет или обязательства. Факты из question не подтверждаются самим вопросом. Явное исправление имеет приоритет, неразрешимый конфликт оставляет поле пустым.

Дополнительный guard проверяет числа, email, URL, handles и известные названия технологий относительно `description` и строк `answer`. Это эвристика с возможными ложными срабатываниями и пропусками; она не доказывает полную смысловую достоверность. Отдельно нужны [live semantic evaluations](../testdata/evaluation_cases.json) и проверка человеком. Fake-provider test доказывает поведение кода при заданном ответе, а не качество настоящей модели.

Конкретное консервативное срабатывание: вопрос «Срок — 2 недели?» и ответ «Да» могут однозначно подтверждать срок для человека, но guard не использует `question` как источник числа. Если `2` отсутствует в `description`/`answer`, поле с этим числом будет отклонено, а после исчерпания попыток вернётся ручной fallback. Явный ответ «Да, 2 недели» содержит проверяемое число в допустимом источнике. Это сознательная граница простой проверки, а не повод автоматически доверять всем фактам из вопросов.

Второй пример ложного срабатывания: вход «две недели», вывод «2 недели». Смысл совпадает, но guard проверяет буквальное наличие цифровых значений и не переводит числительные в цифры; такой вывод может привести к fallback. Человек может сохранить исходную формулировку при ручном заполнении. Эти ограничения не устраняются расширением списка источников на неподтверждённый текст вопросов.

## Rating: найденная реализация backend

AI не рассчитывает score и не меняет rating module. В `origin/backend` (`ba61fef`) прочитан `backend/internal/rating/rating.go`; фактические веса реализации:

| Категория | Вес |
|---|---:|
| Context + need | 10 + 10, независимо |
| Data | 20 |
| Expected result | 15 |
| Success criteria | 15 |
| Constraints | 10 |
| Users | 10 |
| Business connection | contact 5 + interaction_format 5 |
| Всего | 100 |

`filled(value)` означает `strings.TrimSpace(value) != ""`; «не знаю» тоже считается заполнением. `title` и `topic` на score не влияют. Уровни: меньше 40 — `draft`, 40–69 — `working`, 70–89 — `ready`, от 90 — `priority`. Fallback заполняет только context и потому по коду может дать 10 баллов; это не доказательство готовности задачи. Эти сведения получены чтением исходников, совместный запрос AI → сохранение → rating не выполнялся. Политику для «не знаю» и содержательности нужно согласовать с backend; эта AI-поставка её не меняет. В demo показывайте рост только после реального сохранения и чтения backend-результата.

## Приёмка интеграции

1. Запустить `go test ./...` и `go vet ./...`; зафиксировать фактические результаты.
2. Выполнить PowerShell-запросы из [README](../README.md) без ключа, проверить mode и схему.
3. С доступным API отдельно выполнить [evaluation cases](../testdata/evaluation_cases.json), сохраняя вывод как live только при `X-AI-Mode: live`.
4. Подключить frontend с редактированием и подтверждением, затем проверить сохранение и согласованный rating.
5. Выполнить [полный checklist](integration-checklist.md) на существующих компонентах платформы. Mock-тесты не заменяют эти шаги.

При разработке изолированного стенда 23.09.2026 базовый live smoke прошёл на `gpt-4.1-mini`: RU questions и KK card, HTTP 200, `X-AI-Mode: live`, по одной попытке. Секрет и `.local/` в клон не включены; при публикации live-запросы не повторялись. Полные semantic evaluations и совместный platform end-to-end не выполнены. Исторические результаты — [test-results.md](test-results.md), demo-сценарий — [demo.md](demo.md).
