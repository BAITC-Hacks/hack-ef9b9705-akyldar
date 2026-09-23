# Проверка полного сценария

Текущий репозиторий объединяет `backend/`, `ai/` и `frontend/`. Backend подключает AI-модуль к общему `net/http` серверу, а frontend использует эти маршруты через `VITE_API_URL`.

## Фактический статус

| Проверка | Статус | Подтверждение |
|---|---|---|
| AI unit/integration tests | passed | `go test -count=1 ./...` из `ai/` |
| AI static analysis | passed | `go vet ./...` из `ai/` |
| Backend tests | passed | `go test -count=1 ./...` из `backend/` |
| Backend static analysis | passed | `go vet ./...` из `backend/` |
| Frontend build | passed | `npm run build` из `frontend/` |
| Fallback AI without key | passed | backend integration test проверяет `X-AI-Mode: fallback` |
| CORS and AI mode exposure | passed | backend CORS tests and combined handler test |
| Full HTTP MVP flow | passed | `backend/cmd/server/main_test.go:TestCompleteMVPFlow` |
| Live provider semantic evaluation | not run | requires a real provider key and manual review |

## Основной flow

`TestCompleteMVPFlow` использует временную SQLite-базу и `httptest.NewServer`, поэтому не изменяет production database:

1. создаёт слабую задачу;
2. получает минимум три fallback-вопроса;
3. получает fallback-карточку из 11 полей;
4. сохраняет карточку через `PUT` и получает рейтинг `10/draft`;
5. дополняет карточку и получает `100/priority`;
6. получает рейтинг отдельным endpoint;
7. подтверждает и публикует задачу;
8. находит её в каталоге;
9. создаёт предложение существующей команде;
10. принимает предложение через `PATCH` и подтверждает статус повторным `GET`.

Рейтинг в этом flow возвращается backend; тест не подставляет искусственный результат.

## AI API

- `POST /api/ai/questions` возвращает 3–5 непустых вопросов.
- `POST /api/ai/card` возвращает ровно 11 строковых полей карточки.
- Fallback сохраняет исходное описание в `context`, а неизвестные поля оставляет пустыми.
- `X-AI-Mode` различает `live` и `fallback`; frontend показывает режим пользователю.
- Ошибки провайдера не раскрывают ключ, URL или сырой ответ внешнего API.

## Ограничения проверки

Live semantic cases не запускались, потому что для них нужен внешний API-ключ. Они не являются обязательными для локального fallback-demo. Полный сценарий frontend/backend проверен через реальные HTTP-маршруты общего handler; визуальная проверка браузера остаётся ручным шагом.
