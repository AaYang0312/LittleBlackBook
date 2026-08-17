#!/usr/bin/env bash
# 全链路验收：注册→登录→上传→发笔记→关注→点赞(双击幂等)→评论→feed→重建
set -euo pipefail
BASE="${BASE:-http://localhost:8080/api/v1}"
SUFFIX="${RANDOM}"
ALICE="alice${SUFFIX}"
BOB="bob${SUFFIX}"

req() { # method path token json
  local method=$1 path=$2 token=${3:-} body=${4:-}
  local args=(-s -X "$method" "$BASE$path" -H 'Content-Type: application/json')
  [ -n "$token" ] && args+=(-H "Authorization: Bearer $token")
  [ -n "$body" ] && args+=(-d "$body")
  curl "${args[@]}"
}

echo "== 注册/登录 =="
req POST /users/register "" "{\"username\":\"$ALICE\",\"password\":\"123456\"}" | jq -e '.code==0' >/dev/null
req POST /users/register "" "{\"username\":\"$BOB\",\"password\":\"123456\"}" | jq -e '.code==0' >/dev/null
AT=$(req POST /users/login "" "{\"username\":\"$ALICE\",\"password\":\"123456\"}" | jq -r '.data.token')
BT=$(req POST /users/login "" "{\"username\":\"$BOB\",\"password\":\"123456\"}" | jq -r '.data.token')
ALICE_ID=$(req GET /users/me "$AT" | jq -r '.data.id')
BOB_ID=$(req GET /users/me "$BT" | jq -r '.data.id')
[ -n "$AT" ] && [ -n "$BT" ] && [ -n "$ALICE_ID" ] && [ "$ALICE_ID" != "null" ]

echo "== 关注 =="
req POST "/users/$ALICE_ID/follow" "$BT" | jq -e '.code==0' >/dev/null

echo "== 上传图片/发笔记 =="
head -c 100 /dev/urandom > /tmp/e2e.jpg
IMG=$(curl -s -X POST "$BASE/notes/images" -H "Authorization: Bearer $AT" -F "file=@/tmp/e2e.jpg" | jq -r '.data.url')
NOTE_ID=$(req POST /notes "$AT" "{\"title\":\"e2e\",\"content\":\"hello\",\"images\":[\"$IMG\"]}" | jq -r '.data.id')
[ -n "$NOTE_ID" ] && [ "$NOTE_ID" != "null" ]

# 等 fanout 消费者处理
sleep 1

echo "== 点赞（双击幂等） =="
req POST "/notes/$NOTE_ID/like" "$BT" | jq -e '.code==0' >/dev/null
req POST "/notes/$NOTE_ID/like" "$BT" | jq -e '.code==0' >/dev/null   # 双击
sleep 1                                                              # 等 worker 落库
LC=$(req GET "/notes/$NOTE_ID" "" | jq -r '.data.like_count')
[ "$LC" = "1" ] || { echo "like_count=$LC, want 1"; exit 1; }

echo "== 评论 =="
req POST "/notes/$NOTE_ID/comments" "$BT" '{"content":"e2e comment"}' | jq -e '.code==0' >/dev/null
# 同步写，不用 sleep
CC=$(req GET "/notes/$NOTE_ID" "" | jq -r '.data.comment_count')
[ "$CC" = "1" ] || { echo "comment_count=$CC, want 1"; exit 1; }

echo "== 关注页 feed =="
HIT=$(req GET "/feed/following?size=20" "$BT" | jq --argjson id "$NOTE_ID" '[.data[] | select(.id==$id)] | length')
[ "$HIT" = "1" ] || { echo "feed missing note $NOTE_ID"; exit 1; }

echo "== 修改资料/头像 =="
req PUT /users/me "$AT" '{"nickname":"AliceNew","bio":"hi"}' | jq -e '.code==0' >/dev/null
NICK=$(req GET /users/me "$AT" | jq -r '.data.nickname')
[ "$NICK" = "AliceNew" ] || { echo "nickname=$NICK, want AliceNew"; exit 1; }
AV=$(curl -s -X POST "$BASE/users/me/avatar" -H "Authorization: Bearer $AT" -F "file=@/tmp/e2e.jpg" | jq -r '.data.url')
[ -n "$AV" ] && [ "$AV" != "null" ] || { echo "avatar upload failed"; exit 1; }
req PUT /users/me "$AT" "{\"avatar_url\":\"$AV\"}" | jq -e '.code==0' >/dev/null

echo "== 我的笔记 =="
MINE=$(req GET "/users/$ALICE_ID/notes" "" | jq -e '.code==0 and ([.data.list[] | select(.id=='"$NOTE_ID"')] | length == 1)')
[ "$MINE" = "true" ] || { echo "my-notes missing NOTE_ID=$NOTE_ID"; exit 1; }
AUTHOR_NICK=$(req GET /notes/latest "" | jq -r '.data.list[0].author.nickname')
[ "$AUTHOR_NICK" != "null" ] && [ -n "$AUTHOR_NICK" ] || { echo "note author empty: $AUTHOR_NICK"; exit 1; }

echo "== 楼中楼 =="
TOP=$(req POST "/notes/$NOTE_ID/comments" "$AT" '{"content":"top comment"}' | jq -r '.data.id')
[ -n "$TOP" ] && [ "$TOP" != "null" ] || { echo "create top comment failed"; exit 1; }
req POST "/notes/$NOTE_ID/comments" "$AT" "{\"content\":\"reply1\",\"parent_id\":$TOP,\"reply_to\":$ALICE_ID}" | jq -e '.code==0' >/dev/null
req POST "/notes/$NOTE_ID/comments" "$AT" "{\"content\":\"reply2\",\"parent_id\":$TOP,\"reply_to\":$BOB_ID}" | jq -e '.code==0' >/dev/null
RC=$(req GET "/notes/$NOTE_ID/comments" "$AT" | jq --argjson id "$TOP" '[.data[] | select(.id==$id) | .reply_count][0]')
[ "$RC" = "2" ] || { echo "top reply_count=$RC, want 2"; exit 1; }
REPS=$(req GET "/notes/$NOTE_ID/comments/$TOP/replies" "$AT" | jq '.data | length')
[ "$REPS" = "2" ] || { echo "replies=$REPS, want 2"; exit 1; }
REPLY_TO_AUTHOR=$(req GET "/notes/$NOTE_ID/comments/$TOP/replies" "$AT" | jq -r '.data[1].reply_to_author.id')
[ "$REPLY_TO_AUTHOR" = "$BOB_ID" ] || { echo "reply_to_author=$REPLY_TO_AUTHOR, want $BOB_ID"; exit 1; }
TOTAL_CC=$(req GET "/notes/$NOTE_ID" "" | jq -r '.data.comment_count')
[ "$TOTAL_CC" = "4" ] || { echo "comment_count=$TOTAL_CC, want 4 (1 e2e + 1 top + 2 replies)"; exit 1; }
# 不能回复一个回复
req POST "/notes/$NOTE_ID/comments" "$AT" "{\"content\":\"bad\",\"parent_id\":$(req GET "/notes/$NOTE_ID/comments/$TOP/replies" "$AT" | jq -r '.data[0].id')}" | jq -e '.code!=0' >/dev/null

echo "== rebuild-counts =="
curl -s -X POST "$BASE/internal/rebuild-counts" | jq -e '.code==0' >/dev/null

echo "E2E PASS (alice_id=$ALICE_ID bob_id=$BOB_ID note_id=$NOTE_ID)"
