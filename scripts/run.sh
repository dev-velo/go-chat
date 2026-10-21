#!/usr/bin/env bash
set -euo pipefail

# Configure deployment here. Leave the state bucket empty for local state.
export AWS_REGION="us-east-1"
export TF_VAR_aws_iam_role="arn:aws:iam::123456789012:role/go-chat-deploy"
export TF_STATE_BUCKET=""
export TF_VAR_aws_region="${AWS_REGION}"

# Usage: bash scripts/run.sh <command> [environment] [terraform arguments...]
# Commands: build, fmt, validate, init, plan, apply, destroy, output
exec bash "$(dirname "${BASH_SOURCE[0]}")/entrypoint.sh" "$@"
