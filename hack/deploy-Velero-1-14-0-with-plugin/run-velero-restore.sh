#!/bin/bash

# Variables
BACKUP_NAME=$1
NAMESPACE="test-csi-snapshot"
#RESTORE_NAME="restore-${BACKUP_NAME}-$(date +%Y%m%d%H%M%S)"
RESTORE_NAME=$1
MINIO_ALIAS="myminio"
MINIO_URL="http://127.0.0.1:9000"
MINIO_ACCESS_KEY="minio"
MINIO_SECRET_KEY="minio123"
DOWNLOAD_DIR="/tmp"
VELERO_NAMESPACE="velero"  # Change if your Velero is installed in a different namespace
PORT_FORWARD_PID=""
MINIKUBE_PROFILE="client1"
VELERO_BINARY_PATH="/home/swanand/work/projects/poc/learn_code_folder/velero-plugin-poc/hack/deploy-Velero-1-14-0-with-plugin/velero"

# Function to clean up port forwarding
cleanup() {
    if [ -n "$PORT_FORWARD_PID" ]; then
        kill $PORT_FORWARD_PID
        echo "Port-forwarding process killed."
    fi
}

# Trap script exit to clean up port forwarding
trap cleanup EXIT

# Check if Minio is port-forwarded
if ! lsof -i:9000 &>/dev/null; then
    echo "Minio is not port-forwarded. Setting up port-forward..."
    minikube profile "$MINIKUBE_PROFILE"
    kubectl -n "$VELERO_NAMESPACE" port-forward service/minio 9000:9000 &
    PORT_FORWARD_PID=$!
    # Give port-forwarding some time to set up
    sleep 5

    # Ensure the Minio port-forwarding is established
    if ! lsof -i:9000 &>/dev/null; then
        echo "Failed to establish port-forwarding for Minio. Exiting."
        PORT_FORWARD_PID=""
        exit 1
    fi
else
    echo "Minio is already port-forwarded."
fi

# Set up Minio client
mc alias set "$MINIO_ALIAS" "$MINIO_URL" "$MINIO_ACCESS_KEY" "$MINIO_SECRET_KEY"
if [ $? -ne 0 ]; then
    echo "Failed to set up Minio client. Exiting."
    exit 1
fi

# Create a Velero restore
restore_command="$VELERO_BINARY_PATH restore create ${RESTORE_NAME} --from-backup ${BACKUP_NAME} --include-namespaces ${NAMESPACE}"
echo "Executing: $restore_command"
$restore_command
if [ $? -ne 0 ]; then
    echo "Failed to create Velero restore. Exiting."
    exit 1
fi

# Wait for the restore to complete
echo "Waiting for restore to complete..."
while true; do
    STATUS=$(kubectl -n "$VELERO_NAMESPACE" get restores/"$RESTORE_NAME" -o json | jq -r .status.phase)
    if [ "$STATUS" == "Completed" ] || [ "$STATUS" == "Failed" ] || [ "$STATUS" == "PartiallyFailed" ]; then
        break
    fi
    echo "Current status: $STATUS"
    sleep 10
done

# Get restore details and check for errors/warnings
restore_details=$(kubectl -n "$VELERO_NAMESPACE" get restores/"$RESTORE_NAME" -o json)
if [ $? -ne 0 ]; then
    echo "Failed to get Velero restore details. Exiting."
    exit 1
fi

echo "${restore_details}" | jq .

errors=$(echo "${restore_details}" | jq .status.errors)
warnings=$(echo "${restore_details}" | jq .status.warnings)

# Ensure the variables are integers
errors=${errors:-0}
warnings=${warnings:-0}

# Print the number of errors and warnings
echo "Number of errors: ${errors}"
echo "Number of warnings: ${warnings}"

# Download the restore folder using mc
RESTORE_PATH="${MINIO_ALIAS}/velero/restores/${RESTORE_NAME}"
echo "Downloading restore from ${RESTORE_PATH} to ${DOWNLOAD_DIR}..."
mc cp --recursive "$RESTORE_PATH" "$DOWNLOAD_DIR"

if [ $? -eq 0 ]; then
    echo "Restore download completed. Files are stored in ${DOWNLOAD_DIR}/${RESTORE_NAME}"
else
    echo "Failed to download the restore. Exiting."
    exit 1
fi

# Extract and print log file regardless of errors or warnings
echo "Extracting log file..."
gzip -d "${DOWNLOAD_DIR}/${RESTORE_NAME}/restore-${RESTORE_NAME}-logs.gz"

# Print error and warning messages from the log file
echo "Errors and warnings in the log file:"
grep -E "level=(error|warning)" "${DOWNLOAD_DIR}/${RESTORE_NAME}/restore-${RESTORE_NAME}-logs"

