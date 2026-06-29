#!/usr/bin/env bash
set -euo pipefail

POLL_INTERVAL="${POLL_INTERVAL:-120}"
KHA_SKILL="${KHA_SKILL:?KHA_SKILL env var required}"
PROJECT_REPO_URL="${PROJECT_REPO_URL:?PROJECT_REPO_URL env var required}"

# Configure git HTTPS auth
if [ -n "${GIT_TOKEN:-}" ]; then
  echo "https://oauth2:${GIT_TOKEN}@github.com" > /root/.git-credentials
fi

# Clone project repo on first start
if [ ! -d /workspace/.git ]; then
  git clone "$PROJECT_REPO_URL" /workspace
fi

cd /workspace

# Register kha plugin at project level (not global) on first run
CLAUDE_SETTINGS="/workspace/.claude/settings.json"
if ! grep -q "/opt/kha" "$CLAUDE_SETTINGS" 2>/dev/null; then
  mkdir -p /workspace/.claude
  if [ -f "$CLAUDE_SETTINGS" ]; then
    node -e "
      const fs = require('fs');
      const s = JSON.parse(fs.readFileSync('$CLAUDE_SETTINGS', 'utf8'));
      s.plugins = [...(s.plugins || []), '/opt/kha'];
      fs.writeFileSync('$CLAUDE_SETTINGS', JSON.stringify(s, null, 2));
    "
  else
    echo '{"plugins":["/opt/kha"]}' > "$CLAUDE_SETTINGS"
  fi
fi

echo "kha agent starting: skill=$KHA_SKILL poll=${POLL_INTERVAL}s"

while true; do
  git fetch origin
  git checkout develop
  git pull origin develop

  claude -p "/kha:${KHA_SKILL}" \
    --dangerously-skip-permissions \
    2>&1 | tail -20 || true

  echo "[$(date -u +%H:%M:%SZ)] cycle done, sleeping ${POLL_INTERVAL}s..."
  sleep "$POLL_INTERVAL"
done
