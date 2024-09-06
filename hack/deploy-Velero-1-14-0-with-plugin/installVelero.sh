#!/bin/bash

# Variables
NAMESPACE="velero"
MINIO_DEPLOYMENT_FILE="/home/swanand/work/projects/poc/learn_code_folder/velero-plugin-poc/hack/deploy-Velero-1-14-0-with-plugin/velero-code/examples/minio/00-minio-deployment.yaml"
VELERO_INSTALL_PATH="/home/swanand/work/projects/poc/learn_code_folder/velero-plugin-poc/hack/deploy-Velero-1-14-0-with-plugin/velero"
CREDENTIALS_FILE="/home/swanand/work/projects/poc/learn_code_folder/velero-plugin-poc/hack/deploy-Velero-1-14-0-with-plugin/credentials-velero"
BUCKET_NAME="velero"
PROVIDER="aws"
REGION="minio"
S3_URL="http://minio.velero.svc:9000"
VELERO_PLUGINS="velero/velero-plugin-for-aws:v1.10.0,swanandshende/velero-plugin-poc:latest"
FEATURES="EnableCSI"

# Delete existing Velero namespace
kubectl delete ns "$NAMESPACE"

# Apply Minio deployment
kubectl apply -f "$MINIO_DEPLOYMENT_FILE"

# Install Velero
"$VELERO_INSTALL_PATH" install --wait \
 --provider "$PROVIDER" \
 --bucket "$BUCKET_NAME" \
 --secret-file "$CREDENTIALS_FILE" \
 --features="$FEATURES" \
 --backup-location-config region="$REGION",s3ForcePathStyle="true",s3Url="$S3_URL" \
 --use-volume-snapshots=false \
 --plugins "$VELERO_PLUGINS"

echo "D O N E"

