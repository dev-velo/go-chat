# Infrastructure

## Configuration

Edit region, deployment role, and optional state bucket in `../scripts/run.sh`.
The script exports provider inputs and configures backend role assumption.
Initial AWS credentials must be able to assume the role; the role must have
deployment permissions and, for S3 state, access to the state bucket.
Keep region and role out of tfvars to avoid overriding the script settings.

Terraform automatically loads `terraform.tfvars`. All app settings have
defaults: project `go-chat`, environment `dev`, provider `openai`, model
`gpt-4o-mini`, and generated sign-in credentials.

### LLM API key

Terraform creates a SecureString parameter at
`/<project>/<environment>/llm-api-key` by default. Set these in your ignored local
`terraform.tfvars`:

```hcl
llm_provider = "openai"
llm_model    = "gpt-4o-mini"
llm_api_key  = "YOUR_OPENAI_API_KEY"
```

Use an API key and model ID from the selected provider (`openai` or `anthropic`).
The key can also be supplied securely through `TF_VAR_llm_api_key`.

To reuse an existing SecureString parameter in the deployment account and region
instead of creating one:

```hcl
create_llm_api_key_ssm_parameter = false
llm_api_key_ssm_name             = "/shared/openai/api-key"
```

### Login credentials

By default Terraform generates username/password SecureString parameters at
`/<project>/<environment>/app-user-name` and `.../app-user-pass`.
The outputs `app_user_name` and `app_user_pass` are sensitive:

```sh
just tf output dev -raw app_user_name
just tf output dev -raw app_user_pass
just tf output dev app_url
```

`-raw` prints secrets in plaintext; avoid shared terminals and CI logs.
For existing credentials, set `create_app_auth_ssm_parameters=false`,
`app_user_ssm_name`, and `app_pass_ssm_name`. Credential-value outputs are null
in that mode; retrieve existing credentials through your authorized SSM workflow.
Lambda receives parameter names, not secret environment values.

Set `llm_api_key_kms_key_arn` or `app_auth_kms_key_arn` for customer-managed keys.
Their key policies must permit the execution role; parameter creation also
requires deployment-role encryption permissions.

## Commands and state

```sh
just tf plan
just tf apply
just tf output
just tf destroy
```

Commands default to `dev`; use `just tf plan staging` for another environment.
Other commands: `build`, `fmt`, `validate`, `init`. Apply requires a saved plan.
Destroy retains Terraform's confirmation; CI can explicitly pass
`just tf destroy dev -auto-approve`.

`run.sh` holds settings; `entrypoint.sh` builds the Lambda ZIP and runs Terraform.
Terraform uploads `app/build/bootstrap.zip` directly to Lambda, not the chat
bucket. Preserve the ZIP and saved plan between separate CI plan/apply jobs.
The app handles Function URL requests through the Go Lambda runtime and Gin adapter.

An empty state bucket selects local state at `.state/<environment>/terraform.tfstate`;
otherwise state uses S3. The script generates an ignored `backend.generated.tf`.
Backend metadata and plans are separated by backend type and environment.
Changing backend settings does not migrate state; migrate deliberately before
switching an existing deployment.

**Protect state and saved plans: they contain generated credentials and managed
API keys, even when marked sensitive.** Do not commit them. Local state needs
secure backups. Reusing existing SSM parameters avoids putting their values in
Terraform state.

Destroy deletes chat history (`force_destroy=true`) and managed SSM parameters.
Turning creation flags off also plans deletion of managed parameters; remove
them from Terraform management first if retaining them. Existing parameters
referenced by name are not managed or deleted.
