#!/bin/bash

set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"
RUN_SKILL_TESTS="${RUN_SKILL_TESTS:-1}"
TMP_SUFFIX="$(uuidgen | cut -c1-8 | tr '[:upper:]' '[:lower:]')"
TMP_CONVERSATION_ID=""
TMP_LEGACY_CONVERSATION_ID=""
TMP_SKILL_ONE=""
TMP_SKILL_TWO=""
TMP_UPLOAD_FILE="/tmp/mockinterview-upload-$TMP_SUFFIX.txt"
TMP_SKILL_FILE="/tmp/mockinterview-skill-$TMP_SUFFIX.md"

json_request() {
  local method="$1"
  local url="$2"
  local body="$3"
  curl -s -X "$method" "$url" -H "Content-Type: application/json" -d "$body"
}

assert_equals() {
  local actual="$1"
  local expected="$2"
  local label="$3"
  if [ "$actual" != "$expected" ]; then
    echo "ASSERT_FAIL[$label]: expected '$expected' got '$actual'" >&2
    exit 1
  fi
}

assert_nonempty() {
  local value="$1"
  local label="$2"
  if [ -z "$value" ] || [ "$value" = "null" ]; then
    echo "ASSERT_FAIL[$label]: value is empty" >&2
    exit 1
  fi
}

cleanup() {
  if [ -n "$TMP_CONVERSATION_ID" ]; then
    curl -s -X DELETE "$BASE_URL/api/conversations/$TMP_CONVERSATION_ID" >/dev/null || true
  fi
  if [ -n "$TMP_LEGACY_CONVERSATION_ID" ]; then
    curl -s -X DELETE "$BASE_URL/api/conversations/$TMP_LEGACY_CONVERSATION_ID" >/dev/null || true
  fi
  if [ -n "$TMP_SKILL_ONE" ] && [ -f "skills/$TMP_SKILL_ONE/SKILL.md" ]; then
    rm -f "skills/$TMP_SKILL_ONE/SKILL.md" || true
    rmdir "skills/$TMP_SKILL_ONE" 2>/dev/null || true
  fi
  if [ -n "$TMP_SKILL_TWO" ] && [ -f "skills/$TMP_SKILL_TWO/SKILL.md" ]; then
    rm -f "skills/$TMP_SKILL_TWO/SKILL.md" || true
    rmdir "skills/$TMP_SKILL_TWO" 2>/dev/null || true
  fi
  rm -f "$TMP_UPLOAD_FILE" "$TMP_SKILL_FILE" /tmp/mockinterview_fail_review.json || true
}

trap cleanup EXIT

echo "STEP health/profile/skills"
assert_equals "$(curl -s "$BASE_URL/api/health" | jq -r '.status')" "ok" "health"
curl -s "$BASE_URL/api/profile" | jq -e '.profile.interviewCount >= 0' >/dev/null
curl -s "$BASE_URL/api/skills" | jq -e '.skills | length >= 1' >/dev/null

echo "STEP conversation"
CONV_TITLE="联调Smoke-$TMP_SUFFIX"
RENAMED_TITLE="$CONV_TITLE-已改名"
CONV_JSON="$(json_request POST "$BASE_URL/api/conversations" "{\"title\":\"$CONV_TITLE\"}")"
TMP_CONVERSATION_ID="$(printf '%s' "$CONV_JSON" | jq -r '.id')"
assert_nonempty "$TMP_CONVERSATION_ID" "conversation_id"

PATCH_JSON="$(json_request PATCH "$BASE_URL/api/conversations/$TMP_CONVERSATION_ID" "{\"title\":\"$RENAMED_TITLE\",\"pinned\":true}")"
assert_equals "$(printf '%s' "$PATCH_JSON" | jq -r '.title')" "$RENAMED_TITLE" "conversation_rename"
assert_equals "$(printf '%s' "$PATCH_JSON" | jq -r '.pinned')" "true" "conversation_pin"
assert_equals "$(json_request PATCH "$BASE_URL/api/conversations/$TMP_CONVERSATION_ID" '{"archived":true}' | jq -r '.archived')" "true" "conversation_archive"
assert_equals "$(json_request PATCH "$BASE_URL/api/conversations/$TMP_CONVERSATION_ID" '{"archived":false}' | jq -r '.archived // false')" "false" "conversation_unarchive"

echo "STEP files"
printf 'upload smoke %s\n' "$TMP_SUFFIX" > "$TMP_UPLOAD_FILE"
ARTIFACT_JSON="$(json_request POST "$BASE_URL/api/files" "{\"conversationId\":\"$TMP_CONVERSATION_ID\",\"name\":\"notes-$TMP_SUFFIX.md\",\"contentType\":\"text/markdown\",\"content\":\"# smoke\\nartifact body\"}")"
ARTIFACT_ID="$(printf '%s' "$ARTIFACT_JSON" | jq -r '.id')"
assert_nonempty "$ARTIFACT_ID" "artifact_id"
curl -s "$BASE_URL/api/files?conversationId=$TMP_CONVERSATION_ID" | jq -e '.artifacts | length >= 1' >/dev/null
assert_equals "$(curl -s "$BASE_URL/api/files/$ARTIFACT_ID?content=1" | jq -r '.content')" $'# smoke\nartifact body' "artifact_content"

UPDATED_ARTIFACT_JSON="$(json_request PUT "$BASE_URL/api/files/$ARTIFACT_ID" "{\"name\":\"notes-$TMP_SUFFIX-updated.md\",\"content\":\"# smoke updated\\nartifact body 2\"}")"
assert_equals "$(printf '%s' "$UPDATED_ARTIFACT_JSON" | jq -r '.name')" "notes-$TMP_SUFFIX-updated.md" "artifact_update"

UPLOAD_JSON="$(curl -s -X POST "$BASE_URL/api/files" -F "conversationId=$TMP_CONVERSATION_ID" -F "file=@$TMP_UPLOAD_FILE")"
UPLOAD_ID="$(printf '%s' "$UPLOAD_JSON" | jq -r '.id')"
assert_nonempty "$UPLOAD_ID" "upload_id"
assert_equals "$(curl -s "$BASE_URL/api/files/$UPLOAD_ID?download=1")" "upload smoke $TMP_SUFFIX" "artifact_download"

echo "STEP task/run/review/events"
TASK_JSON="$(json_request POST "$BASE_URL/api/tasks" "{\"conversationId\":\"$TMP_CONVERSATION_ID\",\"title\":\"Smoke Task $TMP_SUFFIX\",\"prompt\":\"请模拟一场 Go agent 开发岗位的技术面试，并在最后给出结构化评分。\",\"artifactIds\":[\"$ARTIFACT_ID\"],\"config\":{\"persona\":\"rigorous\",\"level\":\"中级\",\"focus\":\"generalist\",\"mode\":\"standard\",\"timeBudget\":\"15 分钟\",\"outputStyle\":\"interview_plus_score\"},\"modelConfig\":{}}")"
TASK_ID="$(printf '%s' "$TASK_JSON" | jq -r '.id')"
assert_nonempty "$TASK_ID" "task_id"
assert_equals "$(printf '%s' "$TASK_JSON" | jq -r '.modelConfig.apiKey // empty')" "" "task_api_key_hidden"

RUN_JSON="$(json_request POST "$BASE_URL/api/runs" "{\"taskId\":\"$TASK_ID\",\"prompt\":\"请开始面试，并尽量快速给出第一轮问题。\"}")"
RUN_ID="$(printf '%s' "$RUN_JSON" | jq -r '.id')"
assert_nonempty "$RUN_ID" "run_id"

RUN_STATUS=""
for attempt in $(seq 1 15); do
  RUN_DETAIL_JSON="$(curl -s "$BASE_URL/api/runs/$RUN_ID")"
  RUN_STATUS="$(printf '%s' "$RUN_DETAIL_JSON" | jq -r '.run.status')"
  if [ "$RUN_STATUS" != "created" ]; then
    break
  fi
  sleep 2
done
assert_nonempty "$RUN_STATUS" "run_status"
printf '%s' "$RUN_DETAIL_JSON" | jq -e '.events | length >= 1' >/dev/null

CONV_DETAIL_JSON="$(curl -s "$BASE_URL/api/conversations/$TMP_CONVERSATION_ID")"
assert_equals "$(printf '%s' "$CONV_DETAIL_JSON" | jq -r '.tasks[0].id')" "$TASK_ID" "conversation_task_match"
assert_equals "$(printf '%s' "$CONV_DETAIL_JSON" | jq -r '.tasks[0].modelConfig.apiKey // empty')" "" "conversation_api_key_hidden"

echo "STEP cancel/resume"
CONTROL_TASK_JSON="$(json_request POST "$BASE_URL/api/tasks" "{\"conversationId\":\"$TMP_CONVERSATION_ID\",\"title\":\"Control Task $TMP_SUFFIX\",\"prompt\":\"请模拟一场非常详细的 Go agent 面试，并持续输出多轮高密度问题。\",\"config\":{\"persona\":\"rigorous\",\"level\":\"中级\",\"focus\":\"generalist\",\"mode\":\"standard\",\"timeBudget\":\"45 分钟\",\"outputStyle\":\"interview_plus_score\"},\"modelConfig\":{}}")"
CONTROL_TASK_ID="$(printf '%s' "$CONTROL_TASK_JSON" | jq -r '.id')"
CONTROL_RUN_JSON="$(json_request POST "$BASE_URL/api/runs" "{\"taskId\":\"$CONTROL_TASK_ID\",\"prompt\":\"请开始一场尽量详细的面试。\"}")"
CONTROL_RUN_ID="$(printf '%s' "$CONTROL_RUN_JSON" | jq -r '.id')"
sleep 1
CANCEL_JSON="$(curl -s -X POST "$BASE_URL/api/runs/$CONTROL_RUN_ID/cancel")"
CANCEL_STATUS="$(printf '%s' "$CANCEL_JSON" | jq -r '.status // empty')"
if [ "$CANCEL_STATUS" = "cancelled" ]; then
  RESUME_JSON="$(json_request POST "$BASE_URL/api/runs/$CONTROL_RUN_ID/resume" '{"message":"继续，直接进入下一问。","config":{"persona":"rigorous","level":"中级","focus":"generalist","mode":"standard","timeBudget":"45 分钟","outputStyle":"interview_plus_score"},"artifactIds":[]}')"
  RESUME_STATUS="$(printf '%s' "$RESUME_JSON" | jq -r '.status')"
  if [ "$RESUME_STATUS" != "resuming" ] && [ "$RESUME_STATUS" != "running" ]; then
    echo "ASSERT_FAIL[resume_status]: expected resuming/running got '$RESUME_STATUS'" >&2
    exit 1
  fi
fi

echo "STEP legacy interview"
LEGACY_JSON="$(json_request POST "$BASE_URL/api/interview" '{"prompt":"请模拟一场很短的 Go 面试，只问一个问题并结束。","config":{"persona":"supportive","level":"初级","focus":"generalist","mode":"standard","timeBudget":"15 分钟","outputStyle":"interview_plus_score"}}')"
TMP_LEGACY_CONVERSATION_ID="$(printf '%s' "$LEGACY_JSON" | jq -r '.conversationId')"
LEGACY_RUN_ID="$(printf '%s' "$LEGACY_JSON" | jq -r '.runId')"
assert_nonempty "$TMP_LEGACY_CONVERSATION_ID" "legacy_conversation_id"
assert_nonempty "$LEGACY_RUN_ID" "legacy_run_id"
LEGACY_REVIEW_JSON="$(curl -s "$BASE_URL/api/runs/$LEGACY_RUN_ID/review")"
assert_equals "$(printf '%s' "$LEGACY_REVIEW_JSON" | jq -r '.review.runId')" "$LEGACY_RUN_ID" "legacy_review_run_id"
echo "CHECK strategy snapshot"
printf '%s' "$LEGACY_REVIEW_JSON" | jq -e '.review.decision.decision.explanation | strings | length > 0' >/dev/null
printf '%s' "$LEGACY_REVIEW_JSON" | jq -e '.review.decision.state.lastDecision.explanation | strings | length > 0' >/dev/null
printf '%s' "$LEGACY_REVIEW_JSON" | jq -e '(.review.decision.state.history // []) | type == "array"' >/dev/null
printf '%s' "$LEGACY_REVIEW_JSON" | jq -e '(.review.decision.profileFocus // []) | type == "array"' >/dev/null
echo "CHECK trace tree"
printf '%s' "$LEGACY_REVIEW_JSON" | jq -e '.review.trace.questionCount >= 1' >/dev/null
printf '%s' "$LEGACY_REVIEW_JSON" | jq -e '.review.trace.nodes[0].question | strings | length > 0' >/dev/null
printf '%s' "$LEGACY_REVIEW_JSON" | jq -e '.review.trace.nodes[0].explanation | strings | length > 0' >/dev/null
printf '%s' "$LEGACY_REVIEW_JSON" | jq -e '(.review.trace.nodes[0].focusHits // []) | type == "array"' >/dev/null
printf '%s' "$LEGACY_REVIEW_JSON" | jq -e '(.review.trace.nodes[0].weakSignals // []) | type == "array"' >/dev/null
echo "CHECK profile snapshot"
printf '%s' "$LEGACY_REVIEW_JSON" | jq -e '.review.profile.interviewCount >= 0' >/dev/null
printf '%s' "$LEGACY_REVIEW_JSON" | jq -e '(.review.profile.dimensions // []) | type == "array"' >/dev/null
printf '%s' "$LEGACY_REVIEW_JSON" | jq -e '(.review.profile.recommendedFocus // []) | type == "array"' >/dev/null
echo "CHECK scorecard/summary"
printf '%s' "$LEGACY_REVIEW_JSON" | jq -e '((.review.scorecard.summary // .review.summary.decisionExplanation) | strings | length > 0)' >/dev/null
printf '%s' "$LEGACY_REVIEW_JSON" | jq -e '.review.summary.decisionExplanation | strings | length > 0' >/dev/null
printf '%s' "$LEGACY_REVIEW_JSON" | jq -e '(.review.summary.historicalWeaknessesHit // []) | type == "array"' >/dev/null
LEGACY_EVENTS_OUTPUT="$(curl -sN "$BASE_URL/api/runs/$LEGACY_RUN_ID/events")"
printf '%s' "$LEGACY_EVENTS_OUTPUT" | grep -q 'decision.generated'

echo "STEP negative cases"
FAIL_REVIEW_CODE="$(curl -s -o /tmp/mockinterview_fail_review.json -w '%{http_code}' "$BASE_URL/api/runs/not-a-real-run-id/review")"
assert_equals "$FAIL_REVIEW_CODE" "404" "review_404"

echo "STEP optional skill write path"
if [ "$RUN_SKILL_TESTS" = "1" ]; then
  TMP_SKILL_ONE="smoke-skill-$TMP_SUFFIX"
  TMP_SKILL_TWO="smoke-upload-$TMP_SUFFIX"
  CREATE_SKILL_JSON="$(json_request POST "$BASE_URL/api/skills" "{\"name\":\"$TMP_SKILL_ONE\",\"description\":\"smoke create\",\"focusAreas\":[\"observability\"],\"content\":\"# Smoke Skill\\n\\n用于联调测试。\"}")"
  assert_equals "$(printf '%s' "$CREATE_SKILL_JSON" | jq -r '.name')" "$TMP_SKILL_ONE" "skill_create"
  assert_equals "$(curl -s "$BASE_URL/api/skills/$TMP_SKILL_ONE" | jq -r '.skill.description')" "smoke create" "skill_get"
  UPDATE_SKILL_JSON="$(json_request PUT "$BASE_URL/api/skills/$TMP_SKILL_ONE" "{\"description\":\"smoke updated\",\"content\":\"# Smoke Skill\\n\\n更新后的内容。\"}")"
  assert_equals "$(printf '%s' "$UPDATE_SKILL_JSON" | jq -r '.name')" "$TMP_SKILL_ONE" "skill_update"

  cat > "$TMP_SKILL_FILE" <<MARKDOWN
---
name: $TMP_SKILL_TWO
description: smoke upload
focusAreas:
  - retries
---

# Uploaded Smoke Skill

用于上传联调测试。
MARKDOWN

  UPLOAD_SKILL_JSON="$(curl -s -X POST "$BASE_URL/api/skills" -F "file=@$TMP_SKILL_FILE")"
  assert_equals "$(printf '%s' "$UPLOAD_SKILL_JSON" | jq -r '.name')" "$TMP_SKILL_TWO" "skill_upload"
fi

echo "RESULT base_url=$BASE_URL run_status=$RUN_STATUS run_id=$RUN_ID conversation_id=$TMP_CONVERSATION_ID legacy_run_id=$LEGACY_RUN_ID"
