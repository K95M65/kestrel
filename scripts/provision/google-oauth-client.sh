#!/usr/bin/env bash
# Google's "TVs and Limited Input" OAuth client type is console-only — gcloud
# cannot mint it. This script checks login, prints the console URL, and reminds
# you where to paste the id+secret (Device → Channels, or GOOGLE_OAUTH_CLIENT_ID
# / GOOGLE_OAUTH_CLIENT_SECRET on the robot).
set -euo pipefail
PROJECT="${GOOGLE_CLOUD_PROJECT:-${1:-}}"
if [[ -z "$PROJECT" ]]; then
  echo "usage: $0 <gcp-project-id>   (or set GOOGLE_CLOUD_PROJECT)" >&2
  exit 2
fi
if ! command -v gcloud >/dev/null 2>&1; then
  echo "Install gcloud, then: gcloud auth login" >&2
  exit 1
fi
if ! gcloud auth list --filter=status:ACTIVE --format='value(account)' | grep -q .; then
  echo "Run: gcloud auth login" >&2
  echo "Then re-run: $0 $PROJECT" >&2
  exit 1
fi
echo "Project $PROJECT"
echo
echo "1. Open:"
echo "   https://console.cloud.google.com/apis/credentials/oauthclient?project=$PROJECT"
echo "2. Application type: TVs and Limited Input devices"
echo "   Name: Kestrel"
echo "3. Enable APIs (library): Gmail API, Google Calendar API, Google Drive API"
echo "4. OAuth consent screen: External, test users = your Google account"
echo "5. Paste client ID + secret on the robot: Device → Channels → Sign in with Google"
echo "   or export GOOGLE_OAUTH_CLIENT_ID / GOOGLE_OAUTH_CLIENT_SECRET on os-server."
echo
echo "gcloud cannot create this client type. The console is the working path."
