#!/bin/bash

GEN=$(gcloud run services describe --platform=managed --project=$GOOGLE_CLOUD_PROJECT --region=$GOOGLE_CLOUD_REGION --format='value(status.observedGeneration)' $K_SERVICE)
gcloud run services update --platform=managed --project=$GOOGLE_CLOUD_PROJECT --region=$GOOGLE_CLOUD_REGION --update-env-vars=GEN=$GEN $K_SERVICE
