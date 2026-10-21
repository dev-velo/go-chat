resource "random_id" "suffix" {
  byte_length = 4
}

locals {
  name        = "${var.project}-${var.environment}"
  bucket_name = "${var.project}-${var.environment}-${random_id.suffix.hex}"
  bucket_path = "exchanges/"
}

module "exchanges_s3_bucket" {
  source  = "terraform-aws-modules/s3-bucket/aws"
  version = "5.17.0"

  bucket        = local.bucket_name
  force_destroy = true

  block_public_acls        = true
  block_public_policy      = true
  ignore_public_acls       = true
  restrict_public_buckets  = true
  control_object_ownership = true
  object_ownership         = "BucketOwnerEnforced"

  server_side_encryption_configuration = {
    rule = {
      bucket_key_enabled = true
      apply_server_side_encryption_by_default = {
        sse_algorithm = "AES256"
      }
    }
  }

  lifecycle_rule = [
    {
      id                                     = "abort-incomplete-uploads"
      enabled                                = true
      abort_incomplete_multipart_upload_days = 1
    },
    {
      id      = "expire-objects"
      enabled = true
      filter = {
        prefix = local.bucket_path
      }
      expiration = {
        days = 30
      }
    },
  ]

  attach_policy = true
  policy        = data.aws_iam_policy_document.bucket_policy.json
}
