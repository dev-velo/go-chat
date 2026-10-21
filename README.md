# Go Chat

The Go application lives in `app/`. Its HTML frontend is in `app/web/index.html`
and is embedded into the binary at build time. 

Starting with Homebrew installed, run from the repository root:

```sh
brew install just 
just install # install go + download deps
just run
```

`just run` will start the server and open browser on http://localhost:8080.

## Deploying infrastructure

Configure the region, deployment role ARN and optional state bucket at the top
of `scripts/run.sh`. Use existing AWS credentials that can assume that role;
no profile configuration is required by the scripts.

```sh
bash scripts/run.sh plan dev
bash scripts/run.sh apply dev
```

Copy `infra/terraform.tfvars.example` to `infra/terraform.tfvars` for app-specific
settings. `run.sh` supplies deployment settings; `entrypoint.sh` builds the Lambda
ZIP and runs Terraform.
Other commands are `build`, `fmt`, `validate`, `init`, `destroy` and `output`.
Apply requires a saved plan; destroy uses Terraform's normal confirmation.

An empty `TF_STATE_BUCKET` selects local state at
`infra/.state/<environment>/terraform.tfstate`. A nonempty bucket selects S3
state with role assumption. The entrypoint writes an ignored
`infra/backend.generated.tf` to select the backend. Local state contains secrets:
protect and back it up. Changing backend settings does not migrate existing
state; migrate deliberately before changing an existing deployment's backend.

## S3 exchange storage

Set `STORE_BACKEND=s3`, `S3_BUCKET_NAME` to an existing bucket, and
`S3_BUCKET_PATH` to the exchange folder (for example `exchanges/`). The path is
optional; an empty path uses the bucket root. A trailing slash is added to
nonempty paths so adjacent prefixes are not included.

Locally, configure `AWS_REGION` and your AWS profile or credentials. Lambda
uses its execution role. It needs `s3:PutObject` and `s3:GetObject` for the
prefix, plus `s3:ListBucket` scoped to that prefix. Terraform supplies these
permissions and environment variables for Lambda.

Each exchange is a separate AES256-encrypted JSON object with a random key.
History reads all JSON objects under the prefix, follows all listing pages,
and sorts by exchange timestamp before applying the requested limit. Use a
dedicated prefix; this simple implementation is intended for small histories,
not large datasets. Bucket lifecycle rules control retention.

## POC authentication

All routes, including the HTML page, API and health endpoint, require HTTP
Basic Auth. The browser displays its native login prompt; `index.html` is
unchanged. Use HTTPS outside localhost: Basic Auth encodes credentials but does
not encrypt them. This is one shared POC account, not a multi-user system.

`STORE_BACKEND` selects both storage and the source of secrets:

| Backend | Secrets |
|---|---|
| `memory` | Require `LLM_API_KEY`, `APP_USER_NAME`, `APP_USER_PASS` in the environment. |
| `s3` | Fetch the parameters named by `SSM_KEY_NAME`, `SSM_USER_KEY_NAME`, `SSM_PASS_KEY_NAME` with decryption at startup. Direct secret environment variables are ignored. |

`SSM_FETCH_KEY` is removed. Missing configuration or any failed/empty SSM
retrieval prevents startup; there is no fallback. Secrets are cached for the
process lifetime, so recycle execution environments after rotation.

Terraform generates a random username and password in SecureString parameters
by default, at `/<project>/<environment>/app-user-name` and
`/<project>/<environment>/app-user-pass`. Retrieve them through SSM using an
authorized administrator identity to log in; the Lambda receives only parameter
names in its environment. No credentials are exposed as Terraform outputs.
Generated values are stored in Terraform state and saved plans; protect them.

To use existing credentials, set `create_app_auth_ssm_parameters=false` and
provide `app_user_ssm_name` and `app_pass_ssm_name`. For a customer-managed key,
set `app_auth_kms_key_arn` and permit the Lambda role in its key policy.
Disabling creation after deployment or destroying the stack deletes managed
parameters unless first removed from Terraform management.

Basic Auth does not prevent invocation costs or brute-force attempts against
the public endpoint. It is intended for this POC, not production identity.

## SSM API key parameter

By default Terraform uses an existing SecureString parameter at
`/<project>/<environment>/llm-api-key` without reading or managing its value.
To use your own existing parameter, set:

```hcl
llm_api_key_ssm_name = "/shared/openai/api-key"
```

The parameter must be in the deployment account and region. Supply its name,
not its ARN. Lambda's `SSM_KEY_NAME` and scoped IAM permissions use the same
resolved name.

To have Terraform create and manage the parameter, set
`create_llm_api_key_ssm_parameter = true` and supply the nonempty secret through
`TF_VAR_llm_api_key` using your secret-management workflow. The name override
also applies to creation. Do not enable creation for a parameter that already
exists unless you first import it into Terraform.

**Terraform-managed secret values are stored in state and saved plans**, even
though marked sensitive. Protect those artifacts and never commit the key.
Leave creation disabled to keep the secret out of Terraform state. A managed
parameter is deleted on destroy; changing the creation flag back to false also
plans its deletion, so remove it from Terraform management first if retaining it.

For a customer-managed KMS key, set `llm_api_key_kms_key_arn` and ensure its key
policy permits the Lambda role to decrypt. When creating a parameter, the
deployment role also needs permission to encrypt using that key.
