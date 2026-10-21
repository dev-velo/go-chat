# Go Chat

Go chat POC with browser-native username/password authentication.

## Run locally

Install [just](https://just.systems/) and Go, copy `app/.env.example` to
`app/.env.local`, and set `LLM_API_KEY`, `APP_USER_NAME`, and `APP_USER_PASS`.
Keep `STORE_BACKEND=memory`.

```sh
just run
```

Open http://localhost:8080 and enter your configured credentials.

## Deploy to AWS

Requires Terraform >= 1.9, Go, just, zip, and AWS credentials authorized to assume
your deployment role.

1. Edit `AWS_REGION` and `TF_VAR_aws_iam_role` in `scripts/run.sh`. Leave
   `TF_STATE_BUCKET=""` for local state, or set an existing state bucket.
2. Create `infra/terraform.tfvars`:

```hcl
create_llm_api_key_ssm_parameter = true
llm_api_key                      = "YOUR_OPENAI_API_KEY"
```

3. Deploy and retrieve the URL and generated login credentials:

```sh
just tf plan
just tf apply
just tf output dev app_url
just tf output dev -raw app_user_name
just tf output dev -raw app_user_pass
```

**Current blocker:** the app still needs a Lambda handler adapter before the
deployed URL can serve requests.

Never commit secrets. Terraform state and saved plans contain managed secrets;
`-raw` prints credentials in plaintext. Destroy with `just tf destroy` (deletes
chat history and managed credentials).

Details: [application](app/README.md) | [infrastructure](infra/README.md).
