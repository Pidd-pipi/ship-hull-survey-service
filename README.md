# Ship Hull Survey Service

Captures vessel hull survey findings by zone and tracks review closure.

Set `PORT` to choose a listen port; it defaults to `8080`. From `backend/`, run `go run .`. The API exposes `GET /healthz`, `GET /api/findings`, and `POST /api/findings/{id}/status` with `{"status":"open|reviewed|closed"}`. `/` serves a fetch-based static page.

## Enterprise Layout

```text
.
├── backend/              # Go module, all Go source, static assets, Dockerfile
├── database/README.md
├── output/verification.md
├── prompt.txt
├── runtime_smoke.json    # starts `go run .` with workdir `backend`
├── .env.example
└── .gitignore
```

The health check is `GET /healthz`; hull findings are exposed under `/api/findings`.

## Operations workflow

The service also exposes a survey operations workflow for work-order style records:

- `GET /api/ops/records` — search records (`subject`, `status`, `priority`, `owner`, `page`, `pageSize`)
- `POST /api/ops/records` — create a record (requires `owner` and a `site` label)
- `GET /api/ops/records/{id}` — fetch one record
- `POST /api/ops/records/{id}/transition` — move a record to `queued|active|paused|closed` with optional optimistic `expected` revision
- `GET /api/ops/records/{id}/audit` — audit trail for a record
- `GET /api/ops/snapshot` — status/priority summary

Status transitions follow `queued -> active|closed`, `active -> paused|closed`, `paused -> active|closed`.

## Verification

- `gofmt -w *.go` completed.
- `go build ./...` passed.
- `go test ./...` passed for collection, close, invalid status, and missing-record HTTP paths, plus the operations workflow endpoints.
- Runtime smoke: `PORT=18182 go run .`; health, collection, status update, `/`, and `/app.js` each returned HTTP 200, with `sf-301` changing to `closed`. The process was stopped in the same verification operation and no listener remained.

## Engineering Notes

船体检验流程代码按领域模型、校验、状态转换、并发安全存储、审计事件和 HTTP 生命周期分层。请求会保留请求标识并经过恢复与超时保护；状态写入使用版本校验，错误通过可识别的领域错误返回。

除现有接口回归测试外，项目还保留可复用的分页、过滤、策略、工作流和运行健康能力，便于后续扩展而不把业务规则堆积到处理器中。
