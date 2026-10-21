data "aws_caller_identity" "current" {}
data "aws_partition" "current" {}

data "aws_iam_policy_document" "bucket_policy" {
  statement {
    sid    = "AllowSSLRequestsOnly"
    effect = "Deny"

    principals {
      type        = "*"
      identifiers = ["*"]
    }

    actions = ["s3:*"]

    resources = [
      module.exchanges_s3_bucket.s3_bucket_arn,
      "${module.exchanges_s3_bucket.s3_bucket_arn}/*",
    ]

    condition {
      test     = "Bool"
      variable = "aws:SecureTransport"
      values   = ["false"]
    }
  }

  statement {
    sid    = "DenyOutdatedTLS"
    effect = "Deny"

    principals {
      type        = "*"
      identifiers = ["*"]
    }

    actions = ["s3:*"]

    resources = [
      module.exchanges_s3_bucket.s3_bucket_arn,
      "${module.exchanges_s3_bucket.s3_bucket_arn}/*",
    ]

    condition {
      test     = "NumericLessThan"
      variable = "s3:TlsVersion"
      values   = ["1.2"]
    }
  }

  statement {
    sid    = "DenyUnEncryptedObjectUploads"
    effect = "Deny"

    principals {
      type        = "*"
      identifiers = ["*"]
    }

    actions   = ["s3:PutObject"]
    resources = ["${module.exchanges_s3_bucket.s3_bucket_arn}/*"]

    condition {
      test     = "StringNotEqualsIfExists"
      variable = "s3:x-amz-server-side-encryption"
      values   = ["AES256", "aws:kms"]
    }
  }

  statement {
    sid    = "DenyInfectedObjects"
    effect = "Deny"

    principals {
      type        = "*"
      identifiers = ["*"]
    }
    actions = [
      "s3:GetObject",
      "s3:GetObjectVersion",
    ]

    resources = ["${module.exchanges_s3_bucket.s3_bucket_arn}/*"]
    condition {
      test     = "StringEquals"
      variable = "s3:ExistingObjectTag/STATUS"
      values   = ["INFECTED"]
    }
  }
}

data "aws_iam_policy_document" "iam_policy" {
  statement {
    sid       = "ReadAppAuthCredentials"
    effect    = "Allow"
    actions   = ["ssm:GetParameter"]
    resources = local.app_auth_ssm_arns
  }

  dynamic "statement" {
    for_each = var.app_auth_kms_key_arn == null ? [] : [var.app_auth_kms_key_arn]
    content {
      sid       = "DecryptAppAuthCredentials"
      effect    = "Allow"
      actions   = ["kms:Decrypt"]
      resources = [statement.value]
      condition {
        test     = "StringEquals"
        variable = "kms:ViaService"
        values   = ["ssm.${var.aws_region}.${data.aws_partition.current.dns_suffix}"]
      }
      condition {
        test     = "StringEquals"
        variable = "kms:EncryptionContext:PARAMETER_ARN"
        values   = local.app_auth_ssm_arns
      }
    }
  }

  statement {
    sid       = "ReadLLMAPIKey"
    effect    = "Allow"
    actions   = ["ssm:GetParameter"]
    resources = [local.llm_api_key_ssm_arn]
  }

  dynamic "statement" {
    for_each = var.llm_api_key_kms_key_arn == null ? [] : [var.llm_api_key_kms_key_arn]
    content {
      sid       = "DecryptLLMAPIKey"
      effect    = "Allow"
      actions   = ["kms:Decrypt"]
      resources = [statement.value]
      condition {
        test     = "StringEquals"
        variable = "kms:ViaService"
        values   = ["ssm.${var.aws_region}.${data.aws_partition.current.dns_suffix}"]
      }
      condition {
        test     = "StringEquals"
        variable = "kms:EncryptionContext:PARAMETER_ARN"
        values   = [local.llm_api_key_ssm_arn]
      }
    }
  }

  statement {
    sid       = "ReadExchangeObjects"
    effect    = "Allow"
    actions   = ["s3:GetObject"]
    resources = ["${module.exchanges_s3_bucket.s3_bucket_arn}/${local.bucket_path}*.json"]
  }

  statement {
    sid       = "WriteEncryptedExchangeObjects"
    effect    = "Allow"
    actions   = ["s3:PutObject"]
    resources = ["${module.exchanges_s3_bucket.s3_bucket_arn}/${local.bucket_path}*.json"]

    condition {
      test     = "StringEquals"
      variable = "s3:x-amz-server-side-encryption"
      values   = ["AES256"]
    }
  }

  statement {
    sid       = "ListExchangePrefix"
    effect    = "Allow"
    actions   = ["s3:ListBucket"]
    resources = [module.exchanges_s3_bucket.s3_bucket_arn]

    condition {
      test     = "StringEquals"
      variable = "s3:prefix"
      values   = [local.bucket_path]
    }
  }
}
