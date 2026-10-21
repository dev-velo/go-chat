# Application

The Gin app embeds `web/index.html` into its binary. Run `just run` from the
repository root after copying `.env.example` to `.env.local`. The example uses
the fake LLM provider; select `openai` and a model to make real requests.

## Configuration and authentication

`STORE_BACKEND` selects storage and credential sources:

| Backend | Required configuration |
|---|---|
| `memory` | `LLM_API_KEY`, `APP_USER_NAME`, `APP_USER_PASS` |
| `s3` | `S3_BUCKET_NAME`, `SSM_KEY_NAME`, `SSM_USER_KEY_NAME`, `SSM_PASS_KEY_NAME`; AWS region and credentials |

Both modes require `LLM_PROVIDER` and `LLM_MODEL`. `PORT` defaults to `8080`,
`GIN_MODE` to `release`. `S3_BUCKET_PATH` is optional; empty means bucket root.
Use `.env.example` for local examples; the binary itself does not load dotenv.

S3 mode fetches all three SSM parameters with decryption at startup, ignoring
direct environment secrets. Missing/empty values or retrieval errors prevent
startup. Secrets are cached for the process lifetime; restart after rotation.
`SSM_FETCH_KEY` is no longer used.

All routes, including HTML, API, and health checks, require a session.
Signing in at `/login` sets a signed, HttpOnly cookie; `/logout` clears it.
The browser handles credentials without changes to `index.html`. Use HTTPS
outside localhost. This is a shared POC account with no rate limiter or lockout;
authentication does not prevent Lambda invocation costs.

## Storage

Memory storage disappears on restart. S3 stores each exchange as an independent
AES256-encrypted JSON object with a random key. History follows listing pages,
reads JSON objects, and sorts newest first before limiting. Use a dedicated
prefix and small histories; listing reads all stored exchanges.

AWS uses the SDK's default credential chain. S3 requires prefix-scoped
`GetObject`, `PutObject`, and bucket-level `ListBucket`; SSM requires
`GetParameter` and, for customer-managed keys, appropriate KMS permissions.

## Docker

From the repository root:

```sh
docker build -t go-chat ./app
docker run --rm --read-only --cap-drop=ALL --security-opt=no-new-privileges \
  -p 127.0.0.1:8080:8080 --env-file app/.env.local go-chat
```

Open http://localhost:8080. The env file must use Docker's `KEY=value` syntax
(no `export` or shell quoting). Use `STORE_BACKEND=memory` for local runs.
The scratch image contains only the static binary and HTTPS CA certificates,
runs as UID/GID 65532, and needs no writable filesystem. It has no shell.
Use a TLS-terminating proxy when exposing it beyond localhost.

## Lambda

The deployment script cross-compiles Linux ARM64 `bootstrap` and zips it.
In Lambda, the app uses the Go runtime and Gin's API Gateway v2 adapter to handle
Function URL requests. Locally, it runs the normal HTTP server. Responses are
buffered, not streamed. Both OpenAI and Anthropic providers are supported.

Run application tests with `cd app && go test ./...`.
