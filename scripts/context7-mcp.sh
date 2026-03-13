#!/usr/bin/env bash
set -a # Automatically export all variables
# Load environment variables from .env file next to this script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
[ -f "$SCRIPT_DIR/../.env" ] && source "$SCRIPT_DIR/../.env"
set +a

bunx -y @upstash/context7-mcp --api-key $CONTEXT7_API_KEY
