#!/usr/bin/env bash
set -euo pipefail

KHA_SKILL="${KHA_SKILL:?KHA_SKILL env var required}"
PROJECT_REPO_URL="${PROJECT_REPO_URL:?PROJECT_REPO_URL env var required}"

if [ -n "${GIT_TOKEN:-}" ]; then
  echo "https://oauth2:${GIT_TOKEN}@github.com" > /root/.git-credentials
fi

if [ ! -d /workspace/.git ]; then
  git clone "$PROJECT_REPO_URL" /workspace
fi

cd /workspace

git fetch origin
git checkout develop
git pull origin develop

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

echo "kha: skill=$KHA_SKILL"
exec claude -p "/kha:${KHA_SKILL}" --dangerously-skip-permissions 2>&1
