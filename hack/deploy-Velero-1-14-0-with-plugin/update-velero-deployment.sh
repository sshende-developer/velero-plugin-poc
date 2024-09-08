#!/bin/bash

# Check if a tag is provided
if [ $# -eq 0 ]; then
    echo "Please provide a tag as an argument."
    exit 1
fi

NEW_TAG=$1
DEPLOYMENT_NAME="velero"
NAMESPACE="velero"

# Update the deployment
echo "Updating deployment..."
kubectl patch deployment $DEPLOYMENT_NAME -n $NAMESPACE --type=json -p '[
  {
    "op": "replace",
    "path": "/spec/template/spec/initContainers/1/image",
    "value": "swanandshende/velero-plugin-poc:'$NEW_TAG'"
  },
  {
    "op": "replace",
    "path": "/spec/template/spec/initContainers/1/imagePullPolicy",
    "value": "Always"
  }
]'

# Check if the patch was successful
if [ $? -ne 0 ]; then
    echo "Failed to update the deployment. Exiting."
    exit 1
fi

echo "Deployment updated successfully."

# Restart the deployment
echo "Restarting deployment..."
kubectl rollout restart deployment/$DEPLOYMENT_NAME -n $NAMESPACE

# Check if the restart was successful
if [ $? -ne 0 ]; then
    echo "Failed to restart the deployment. Exiting."
    exit 1
fi

echo "Deployment restart initiated."

# Wait for the rollout to complete
echo "Waiting for rollout to complete..."
kubectl rollout status deployment/$DEPLOYMENT_NAME -n $NAMESPACE --timeout=300s

# Check if the rollout was successful
if [ $? -ne 0 ]; then
    echo "Deployment rollout failed or timed out. Please check the deployment status."
    exit 1
fi

echo "Deployment successfully updated and restarted."
