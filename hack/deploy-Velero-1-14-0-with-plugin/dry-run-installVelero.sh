#!/bin/bash

# Variables
VELERO_BINARY="/home/swanand/work/projects/poc/learn_code_folder/velero-plugin-poc/hack/deploy-Velero-1-14-0-with-plugin/velero"
PROVIDER="aws"
BUCKET_NAME="velero"
SECRET_FILE="/home/swanand/work/projects/poc/learn_code_folder/velero-plugin-poc/hack/deploy-Velero-1-14-0-with-plugin/credentials-velero"
FEATURES="EnableCSI"
REGION="minio"
S3_URL="http://minio.velero.svc:9000"
USE_VOLUME_SNAPSHOTS="false"
PLUGINS="velero/velero-plugin-for-aws:v1.10.0,swanandshende/velero-plugin-poc:latest"
DRY_RUN_OUTPUT_FILE="/home/swanand/work/projects/poc/learn_code_folder/velero-plugin-poc/hack/deploy-Velero-1-14-0-with-plugin/dry-run-velero-install.yaml"

# Run Velero install command with dry-run
$VELERO_BINARY install \
 --provider "$PROVIDER" \
 --bucket "$BUCKET_NAME" \
 --secret-file "$SECRET_FILE" \
 --features="$FEATURES" \
 --backup-location-config region="$REGION",s3ForcePathStyle="true",s3Url="$S3_URL" \
 --use-volume-snapshots="$USE_VOLUME_SNAPSHOTS" \
 --plugins "$PLUGINS" \
 --dry-run --output yaml > "$DRY_RUN_OUTPUT_FILE"

echo "D O N E"
