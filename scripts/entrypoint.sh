#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
COMMAND="${1:-plan}"
ENVIRONMENT="${2:-dev}"
if (($# > 0)); then shift; fi
if (($# > 0)); then shift; fi

case "$COMMAND" in
  build|fmt|validate|init|plan|apply|destroy|output) ;;
  *) echo "Usage: bash scripts/run.sh <build|fmt|validate|init|plan|apply|destroy|output> [environment] [terraform arguments...]" >&2; exit 1 ;;
esac
[[ "$ENVIRONMENT" =~ ^[a-z0-9-]{1,12}$ ]] || {
  echo "Environment must be 1-12 lowercase letters, digits or hyphens." >&2; exit 1;
}
export TF_VAR_environment="$ENVIRONMENT"
BACKEND="local"
if [[ -n "${TF_STATE_BUCKET:-}" ]]; then BACKEND="s3"; fi
PLAN="$ROOT/infra/.plans/$BACKEND/$ENVIRONMENT.tfplan"

build() {
  mkdir -p "$ROOT/app/build"
  (cd "$ROOT/app" && CGO_ENABLED=0 GOOS=linux GOARCH=arm64 \
    go build -trimpath -ldflags="-s -w" -o build/bootstrap ./cmd)
  (cd "$ROOT/app/build" && zip -q -j -FS bootstrap.zip bootstrap)
}

case "$COMMAND" in
  build) build; exit ;;
  fmt) terraform -chdir="$ROOT/infra" fmt -recursive "$@"; exit ;;
esac

# Backend type cannot be selected through -backend-config. Generate only this
# ignored file; never rewrite configuration or remove existing state.
if [[ -n "${TF_STATE_BUCKET:-}" ]]; then
  printf 'terraform {\n  backend "s3" {}\n}\n' > "$ROOT/infra/backend.generated.tf"
  BACKEND_ARGS=(
    "-backend-config=bucket=$TF_STATE_BUCKET"
    "-backend-config=key=${TF_VAR_project:-go-chat}/$ENVIRONMENT/terraform.tfstate"
    "-backend-config=region=${TF_STATE_REGION:-${AWS_REGION:?Set AWS_REGION in run.sh}}"
    "-backend-config=assume_role={role_arn=\"${TF_VAR_aws_iam_role:?Set the deployment role in run.sh}\"}"
    "-backend-config=use_lockfile=true"
  )
else
  printf 'terraform {\n  backend "local" {}\n}\n' > "$ROOT/infra/backend.generated.tf"
  BACKEND_ARGS=("-backend-config=path=$ROOT/infra/.state/$ENVIRONMENT/terraform.tfstate")
fi

if [[ "$COMMAND" == validate ]]; then
  terraform -chdir="$ROOT/infra" init -backend=false -input=false
  terraform -chdir="$ROOT/infra" validate "$@"
  exit
fi

# Separate backend metadata prevents switching environments from reusing the
# previous environment's backend. Switching backend types is not state migration.
export TF_DATA_DIR="$ROOT/infra/.terraform/$BACKEND/$ENVIRONMENT"
mkdir -p "$ROOT/infra/.plans/$BACKEND" "$ROOT/infra/.state/$ENVIRONMENT"
terraform -chdir="$ROOT/infra" init -input=false -reconfigure "${BACKEND_ARGS[@]}"

case "$COMMAND" in
  init) ;;
  plan)
    build
    terraform -chdir="$ROOT/infra" plan -input=false -out="$PLAN" "$@"
    ;;
  apply)
    [[ -f "$PLAN" ]] || { echo "Run plan $ENVIRONMENT first." >&2; exit 1; }
    terraform -chdir="$ROOT/infra" apply -input=false "$@" "$PLAN"
    rm -f "$PLAN"
    ;;
  destroy)
    # Keep Terraform's confirmation; CI must explicitly pass -auto-approve.
    terraform -chdir="$ROOT/infra" destroy "$@"
    rm -f "$PLAN"
    ;;
  output) terraform -chdir="$ROOT/infra" output "$@" ;;
esac
