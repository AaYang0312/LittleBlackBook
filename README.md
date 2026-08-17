# xhs-demo 小红书风格后端 Demo

一个体现**高并发读、海量写计数**场景的社交 Feed 后端 API，覆盖简历/面试核心故事：Feed 流推模式（fan-out on write）、Redis 实时计数 + MQ 异步落库的最终一致性架构、缓存三问题防护、三层幂等设计。

纯后端 API 服务，模块化单体（一个代码库、两个进程），模块边界即未来微服务边界。

## 技术栈

| 组件 | 选型 | 备注 |
|---|---|---|
| 语言/框架 | Go 1.25 + Gin + GORM | |
| 数据库 | MySQL 8 | 关系数据 + 计数持久化（事实源） |
| 缓存 | Redis 7 | timeline、计数、笔记缓存、判重、空值缓存 |
| 消息队列 | RabbitMQ | 点赞事件、feed 分发事件（at-least-once） |
| 对象存储 | MinIO | 笔记图片，S3 兼容 API |
| ID 生成 | 雪花算法（bwmarrin/snowflake） | 笔记/评论 ID |
| 并发原语 | golang.org/x/sync singleflight | 防缓存击穿 |
| 日志 | log/slog | 结构化 JSON，带 request_id |
| 接口文档 | swaggo/swag | 注解生成 Swagger |
| 部署 | docker-compose | MySQL / Redis / RabbitMQ / MinIO |

**MQ 选型理由**：demo 量级下 RabbitMQ 足够，概念简单、运维轻；Kafka 的优势在海量吞吐与消息回溯，属于演进方向。

## 架构

```text
                 ┌────────────┐   feed_fanout    ┌──────────┐
 client ──HTTP──►│ cmd/server  │─────────────────►│ RabbitMQ │──► cmd/worker ──► MySQL(likes/collects 落库)
                 │  (Gin API)  │   like_event     └──────────┘        │
                 └──┬───┬───┬──┘                                      ▼
                    │   │   │                              ZADD feed:inbox:{uid}（推模式）
              MySQL │   │Redis│ ◄── 实时计数 / timeline / 笔记缓存 / 判重
            (事实源)│   │     │
                    │   │MinIO│ ◄── 笔记图片
                    ▼   ▼     ▼
```

- **cmd/server**：Gin HTTP 服务，处理所有读写接口
- **cmd/worker**：独立进程，消费 `feed_fanout`（feed 分发）与 `like_event`（点赞落库，含重试与死信队列 `like_event.dlq`）两个队列
- **内部模块**：`user`（注册/登录/资料）、`note`（发布/详情/发现页/图片上传）、`interaction`（点赞/收藏/评论/关注）、`feed`（关注页 timeline），每模块 `handler → service → repository` 三层
- **模块隔离规则**：禁止跨模块 import 其他模块的 repository，只能调用其 service——拆微服务时把 service 接口换成 RPC 客户端即可

## 快速开始

```bash
make up              # docker compose 一键启动 MySQL/Redis/RabbitMQ/MinIO（自动执行 sql/ 建表）
make run-server      # 启动 HTTP 服务（:8080）
make run-worker      # 启动 MQ worker（另开终端）
make e2e             # 跑全链路验收脚本
```

> 老库升级：执行 `sql/migrations/001_comments_threading.sql`（幂等，可重复执行）；全新库由 `make up` 自动执行 `sql/schema.sql` 建表。

- Swagger 文档：http://localhost:8080/swagger/index.html
- 健康检查：`curl localhost:8080/healthz`
- 核心接口（前缀 `/api/v1`；写操作与个人数据接口需 `Authorization: Bearer <token>`，笔记详情/发现页/用户笔记列表公开可读）：

| 接口 | 说明 |
|---|---|
| `POST /users/register` `POST /users/login` `GET /users/me` `PUT /users/me` `POST /users/me/avatar` | 用户（注册/登录/资料/头像） |
| `POST /notes` `GET /notes/{id}` `GET /notes/latest` `DELETE /notes/{id}` `POST /notes/images` | 笔记 |
| `GET /users/{id}/notes` | 用户笔记列表（游标分页） |
| `POST/DELETE /users/{id}/follow` | 关注/取关 |
| `POST/DELETE /notes/{id}/like` `POST/DELETE /notes/{id}/collect` | 点赞/收藏（Redis 判重幂等） |
| `POST /notes/{id}/comments` `GET /notes/{id}/comments` | 评论（POST 可选 `parent_id`/`reply_to`；GET 仅顶级评论，带 `reply_count`） |
| `GET /notes/{id}/comments/{cid}/replies` | 展开回复（游标分页，正序） |
| `GET /feed/following` | 关注页 Feed（推模式 inbox） |
| `POST /internal/rebuild-counts` | 计数重建（内网接口，无 JWT） |

## 核心设计决策与面试话术

### 1. Feed 推模式（fan-out on write），发现页为何不用 timeline

- 关注页有关注关系作为集合边界，发布时把笔记推入每个粉丝的 `feed:inbox:{uid}`（ZSet，score=发布时间戳，修剪至最新 500 条），读路径只需 `ZREVRANGE` + 批量查详情，**读 O(1)**
- 发现页面向全站笔记池**无边界**，推模式写放大为全站用户数，不可行，因此走 MySQL 游标分页（`id < cursor ORDER BY id DESC`），**不走 Redis timeline**
- 话术："真实方案是推荐系统多路召回 → 粗排 → 精排；本 demo 以全站最新作为召回源占位，接口契约（游标分页）与推荐场景一致，未来可平滑替换。"

### 2. 点赞异步落库 + 三层幂等

写路径（取消点赞对称）：

```text
client → SADD note:like:users:{noteId} userId
  ├─ 返回 1 → INCR note:like:count:{noteId} → 发 MQ like_event
  └─ 返回 0 → 已赞过，直接返回成功（幂等，不发 MQ）
worker 消费: INSERT IGNORE likes → UPDATE notes.like_count+1 → ack
  失败 nack 重试 3 次 → 死信队列
```

| 层次 | 机制 | 防什么 |
|---|---|---|
| Redis 判重 | `SADD` 返回 0 则不发 MQ | 用户双击、前端重试 |
| MQ 消费 | nack 重试 3 次 → 死信队列 | 消费者宕机丢消息 |
| DB 唯一索引 | `uk_user_note` + `INSERT IGNORE` | MQ at-least-once 重复投递 |

话术："RabbitMQ 只保证 at-least-once，幂等必须落在消费端与存储层，不能依赖 MQ 不重复。"

### 3. 缓存三问题

- **穿透**：回源查不到 → 写 `note:notfound:{id}`（TTL 30s），读路径先查空值标记，命中直接 404
- **击穿**：回源必须经 `singleflight.Group`，同一 noteId 并发回源合并为一次 DB 查询
- **雪崩**：TTL 随机抖动（1h ± 10min）；Redis 不可用时读路径降级直查 MySQL + 日志告警

### 4. 计数分级（按操作频率）

- **点赞/收藏（高频写）**：Redis 实时计数 + MQ 异步落库，读永远以 Redis 为准
- **评论（低频写）**：不走 MQ，直插 `comments` + 同步更新计数，简化链路
- 话术："点赞高频走异步削峰，评论低频走同步简化链路，按操作频率分级处理。"

### 5. 计数一致性与重建（rebuild-counts）

- 实时读永远以 Redis 计数为准；**MySQL 关系表（likes/collects/comments）是唯一事实源**
- `POST /internal/rebuild-counts`：遍历全部笔记，从关系表 `COUNT(*)` 重建 Redis 计数与 notes 表计数列，用于 Redis 数据丢失后的恢复（幂等，失败即中断可重跑）

### 6. 评论楼中楼与作者回填（DTO + 服务层 enrichment）

- **两级模型**：`comments.parent_id`（0=顶级，>0=回复），回复只能挂在顶级评论下（"回复的回复"返回参数错误，与小红书一致）；`reply_count` 在创建回复时同步 `+1`，列表页免 count 查询
- **计数语义**：`note.comment_count` 仍为总数（顶级+回复），走原有 `AddCountDelta` + Redis `INCR` 链路；`rebuild-counts` 的 `COUNT(*)` 语义不变
- **作者回填**：`CommentDTO`/`NoteDTO` 带 `author`/`reply_to_author` 快照，服务层收集一页内所有 `user_id`/`reply_to` 去重后一次 `BatchByIDs` 批量查询回填，避免 N+1；跨模块只依赖 `UserLookup` 接口（`user.Service` 结构化满足），不 import 他人 repository
- **快照与实时**：列表/评论不走缓存，作者实时新鲜；笔记 Detail 缓存含作者快照，改资料后最多 1h 才刷新（见已知取舍）
- 话术："实体管持久化、DTO 管对外契约，服务层做聚合回填——模块边界干净、一次批量查询、敏感字段（如 PasswordHash）物理上不出模块。"

## 演进方向

1. **推拉结合**：粉丝数超阈值的作者走拉模式，普通人走推模式，读时合并两路
2. **发现页接推荐系统/ES** 多路召回，替换"全站最新"占位召回
3. **布隆过滤器**替代空值缓存；sentinel/熔断替代简单降级
4. **模块边界即微服务边界**：service 接口换 RPC（go-zero/Kratos + gRPC），user/note/interaction/feed 拆为四个服务
5. **Kafka 替换 RabbitMQ**（海量吞吐、消息回溯）
6. **分库分表**

## 已知取舍（演进项）

| 取舍 | 说明 |
|---|---|
| feed_fanout 无 DLQ | 分发失败直接丢弃日志，粉丝可能漏看笔记；演进：加死信队列 + 对账任务 |
| `/internal/rebuild-counts` 无鉴权 | demo 内网接口，生产应加鉴权/网段限制 |
| MinIO bucket 公共读 | demo 简化，生产应走签名 URL 私有读 |
| `BatchByIDs` 未走缓存批量优化 | 详情批量读直接查库，演进：管道批量回填缓存 |
| Detail 作者快照缓存 staleness | 改资料后详情页作者最多 1h 才刷新；接受。演进：改资料时失效该用户笔记缓存 |
| 单机雪花节点 | 生产多实例需按节点分配 ID 段或引入发号器 |
