# 小红书 Demo（xhs-demo）设计文档

- 日期：2026-07-30
- 定位：简历项目，纯后端 API demo
- 工期：3 周业余（约 45-60 小时）
- 状态：已与作者逐节评审确认

## 1. 项目目标

在已有 Go 单体 CRUD 项目（书城：Gin + GORM + MySQL + Redis + JWT）基础上，做一个体现**高并发读、海量写计数**场景的社交 Feed 后端，覆盖以下简历/面试核心故事：

- Feed 流推模式（fan-out on write）
- Redis 计数 + MQ 异步落库的最终一致性架构
- 缓存三问题（穿透/击穿/雪崩）的防护落地
- 三层幂等设计（Redis 判重 / MQ 重试+死信 / DB 唯一索引）
- 模块化单体，模块边界预留微服务演进

**交付形态**：纯后端 API 服务 + Swagger 文档 + docker-compose 一键启动 + 集成验收脚本。不做前端。

**明确不做**：视频、私信、通知系统、推荐算法、楼中楼评论、管理后台、微服务拆分。

## 2. 技术栈

| 组件 | 选型 | 备注 |
|---|---|---|
| 语言/框架 | Go 1.25 + Gin + GORM | 延续书城技术栈 |
| 数据库 | MySQL 8 | 关系数据 + 计数持久化（事实源） |
| 缓存 | Redis 7 | timeline、计数、笔记缓存、判重、空值缓存 |
| 消息队列 | RabbitMQ | 点赞事件、feed 分发事件（at-least-once） |
| 对象存储 | MinIO | 笔记图片，S3 兼容 API |
| ID 生成 | 雪花算法（bwmarrin/snowflake 或 sonyflake） | 笔记/评论 ID |
| 并发原语 | golang.org/x/sync singleflight | 防缓存击穿 |
| 日志 | log/slog | 结构化 JSON，带 request_id |
| 接口文档 | swaggo/swag | 注解生成 Swagger |
| 部署 | docker-compose | MySQL / Redis / RabbitMQ / MinIO / server / worker |

**MQ 选型理由（面试话术）**：demo 量级下 RabbitMQ 足够，概念简单、运维轻；Kafka 的优势在海量吞吐与消息回溯，属于演进方向。

## 3. 架构与项目结构

模块化单体：一个代码库，两个进程（HTTP server + MQ worker），四个业务模块。模块边界即未来微服务边界。

```text
xhs-demo/
├── cmd/
│   ├── server/              # HTTP 服务入口
│   └── worker/              # MQ 消费者入口（独立进程）
├── internal/
│   ├── user/                # 注册/登录/资料（JWT）
│   ├── note/                # 发布/详情/列表/图片上传
│   ├── interaction/         # 点赞/收藏/评论/关注
│   ├── feed/                # 关注页 timeline / 发现页
│   └── pkg/
│       ├── snowflake/       # 分布式 ID
│       ├── cache/           # singleflight、空值缓存封装
│       ├── errs/            # 统一业务错误码
│       └── mq/              # RabbitMQ 连接与队列声明
├── configs/config.yaml
├── deployments/docker-compose.yml
├── sql/                     # 建表脚本 + 种子数据
├── scripts/                 # 集成验收脚本
├── Makefile
└── README.md                # 架构图 + 演进路线（推拉结合 / ES / 微服务）
```

**分层约定**：每个模块内部 `handler → service → repository` 三层。

**模块隔离规则**：禁止跨模块 import 其他模块的 repository；需要其他模块数据时只能调用其 service。拆微服务时把 service 接口换成 RPC 客户端即可。

**worker 进程**消费两个队列：

- `feed_fanout`：笔记分发事件
- `like_event`：点赞/收藏计数落库事件（含重试与死信队列 `like_event.dlq`）

## 4. 数据模型

### 4.1 MySQL 表（6 张）

```sql
-- 用户
users(
  id PK, username UNIQUE, password_hash, nickname, avatar_url, bio,
  created_at, updated_at
)

-- 笔记；计数列是 worker 异步刷新的最终一致性结果，不是实时准确值
notes(
  id PK,                       -- 雪花 ID
  user_id FK, title, content, cover_url,
  images JSON,                 -- ["minio对象路径1","路径2"]
  like_count, collect_count, comment_count,
  status TINYINT,              -- 0正常 1删除
  created_at, updated_at,
  INDEX idx_user_created(user_id, created_at)
)

-- 点赞关系；唯一索引是幂等最终兜底
likes(
  id PK, user_id, note_id, created_at,
  UNIQUE uk_user_note(user_id, note_id)
)

-- 收藏关系
collects(
  id PK, user_id, note_id, created_at,
  UNIQUE uk_user_note(user_id, note_id)
)

-- 评论（仅一级；parent_id 预留在演进说明中，本版不实现楼中楼）
comments(
  id PK,                       -- 雪花 ID
  note_id, user_id, content, status, created_at,
  INDEX idx_note_created(note_id, created_at)
)

-- 关注关系
follows(
  id PK, follower_id, followee_id, created_at,
  UNIQUE uk_pair(follower_id, followee_id),
  INDEX idx_followee(followee_id)
)
```

### 4.2 Redis Key 设计

| Key | 类型 | 用途 | TTL |
|---|---|---|---|
| `note:cache:{id}` | String(JSON) | 笔记详情缓存 | 1h ± 10min 随机抖动 |
| `note:notfound:{id}` | String | 空值缓存，防穿透 | 30s |
| `note:like:users:{noteId}` | Set | 点赞判重 | 长期 |
| `note:like:count:{noteId}` | String | 实时点赞数 | 长期 |
| `note:collect:users:{noteId}` / `note:collect:count:{noteId}` | 同上模式 | 收藏判重与计数 | 长期 |
| `note:comment:count:{noteId}` | String | 评论计数 | 长期 |
| `feed:inbox:{userId}` | ZSet | 关注页 timeline，score=发布时间戳 | 长期，修剪至最新 500 条 |

### 4.3 MinIO

单 bucket `xhs-images`，对象名 `{userId}/{snowflakeId}.{ext}`。上传流程：客户端 → server 接收 → 转存 MinIO → DB 只存对象路径，接口返回可访问 URL。

## 5. 核心流程时序

### 5.1 发布笔记 → Feed 分发（推模式）

```text
client → note 模块: 上传图片到 MinIO → INSERT notes → 返回成功
                     └→ 发 MQ 消息 feed_fanout {noteId, authorId, ts}
worker 消费 feed_fanout:
  查 follows 找作者全部粉丝
  → 循环 ZADD feed:inbox:{fanId} score=ts member=noteId
  → ZREMRANGEBYRANK 修剪至最新 500 条 → ack
```

HTTP 接口**不等分发完成**即返回成功，分发异步最终一致（削峰话术）。粉丝翻页超出 inbox 500 条范围时返回"没有更多"。

### 5.2 点赞（写路径；取消点赞对称）

```text
client → interaction 模块:
  1. SADD note:like:users:{noteId} userId
     ├─ 返回 1 → INCR note:like:count:{noteId} → 发 MQ like_event{noteId,userId,+1}
     └─ 返回 0 → 已赞过，直接返回成功（幂等，不发 MQ）
worker 消费 like_event:
  INSERT IGNORE likes → UPDATE notes SET like_count=like_count+1 → ack
  消费失败 nack 重试 3 次 → 死信队列
```

### 5.3 读关注页 Feed（读路径）

```text
client → feed 模块: ZREVRANGE feed:inbox:{me} offset offset+size
  → 批量 GET note:cache:{id}
    ├─ 命中 → 直接使用
    └─ 未命中 → singleflight 合并回源 → 批量查 MySQL → 回填缓存
  → 叠加 Redis 实时计数（覆盖 notes 表中的旧值）
```

### 5.4 发现页

`SELECT ... FROM notes WHERE status=0 AND id < :cursor ORDER BY id DESC LIMIT :size`，游标分页，不走 Redis timeline。

**设计理由（面试话术）**：关注页有关注关系作为集合边界，适合推模式 inbox；发现页面向全站笔记池无边界，推模式写放大为全站用户数，不可行。真实方案是推荐系统多路召回（热门/协同过滤/标签）→ 粗排 → 精排，数据源是推荐引擎与 ES 而非收件箱。本 demo 以"全站最新"作为召回源占位，接口契约（游标分页）与推荐场景一致，未来可平滑替换。

### 5.5 评论（按操作频率分级处理）

低频写，不走 MQ：直插 `comments` + 同步 `UPDATE notes.comment_count` + `INCR note:comment:count:{noteId}`。话术："点赞高频走异步削峰，评论低频走同步简化链路，按操作频率分级处理"。

## 6. 缓存三问题与幂等

### 6.1 缓存穿透

回源查不到 → 写 `note:notfound:{id}`（TTL 30s）；读路径先查空值标记，命中直接 404。演进：入口层布隆过滤器。

### 6.2 缓存击穿

回源必须经 `singleflight.Group`，同一 noteId 并发回源合并为一次 DB 查询。

### 6.3 缓存雪崩

- TTL 随机抖动（1h ± 10min）
- Redis 不可用时读路径降级为直查 MySQL + 日志告警（不引入熔断器；演进：sentinel/熔断）

### 6.4 三层幂等

| 层次 | 机制 | 防什么 |
|---|---|---|
| Redis 判重 | `SADD` 返回 0 则不发 MQ | 用户双击、前端重试 |
| MQ 消费 | nack 重试 3 次 → 死信队列 | 消费者宕机丢消息 |
| DB 唯一索引 | `uk_user_note` + `INSERT IGNORE` | MQ at-least-once 重复投递 |

话术："RabbitMQ 只保证 at-least-once，幂等必须落在消费端与存储层，不能依赖 MQ 不重复"。

### 6.5 计数一致性与重建

- 实时读永远以 Redis 计数为准；MySQL 关系表（likes/collects/comments）是唯一事实源
- 提供 `POST /internal/rebuild-counts`：从关系表 `COUNT(*)` 重建 Redis 计数与 notes 计数列，用于 Redis 数据丢失后的恢复

## 7. 错误处理与日志

- `internal/pkg/errs` 统一业务错误码：`1001 参数错误 / 2001 未登录 / 3001 笔记不存在 / 3002 重复点赞 …`
- service 层返回带码错误；handler 中间件统一翻译为 `{code, message, data}` + 对应 HTTP 状态
- 禁止 handler 直接暴露 `err.Error()`（防泄漏 SQL 等内部信息）
- slog 结构化 JSON 日志；middleware 生成 `request_id` 贯穿请求；错误日志带堆栈

## 8. 测试策略

- **单测聚焦 service 层核心逻辑**：点赞幂等、feed 分发、游标分页；repository 以接口 mock
- **集成验收脚本**（`scripts/`）：docker-compose 起依赖后跑通全链路：注册 → 登录 → 发笔记（含图片）→ 关注 → 点赞 → 刷关注页看到该笔记且计数正确
- MQ 消费者测试直接调用消费者函数，绕过真实 RabbitMQ
- 不追求覆盖率数字

## 9. 里程碑（3 周业余，每周约 15-20h）

| 里程碑 | 内容 | 验收标准 |
|---|---|---|
| W1 地基 | 项目骨架、配置/日志/错误码/request_id、docker-compose、用户注册登录 JWT、雪花 ID、Swagger | compose 一键起；注册登录 curl 通；Swagger 可访问 |
| W2 内容 | MinIO 图片上传、笔记 CRUD、发现页游标分页、笔记缓存（穿透/击穿防护） | 发图发笔记全流程通；压测详情接口缓存生效 |
| W3 上半 互动 | 关注/取关、点赞/收藏（Redis 计数 + MQ + 三层幂等）、评论 | 快速双击点赞计数不错；kill worker 重启后消息不丢 |
| W3 下半 Feed+收尾 | feed_fanout 消费者、关注页读链路、rebuild-counts、集成验收脚本、README 架构图与演进话术 | 全链路脚本绿；README 完整 |

**砍功能保险绳**：W3 上半结束时若点赞链路未完成，砍评论保 Feed 链路——Feed 是项目灵魂，评论是锦上添花。

## 10. README 必须包含的演进话术

1. 推拉结合：粉丝数超阈值的作者走拉模式，普通人走推模式，读时合并两路
2. 发现页接推荐系统/ES 多路召回，替换"全站最新"占位召回
3. 布隆过滤器替代空值缓存；sentinel/熔断替代简单降级
4. 模块边界即微服务边界：service 接口换 RPC（go-zero/Kratos + gRPC），user/note/interaction/feed 拆为四个服务
5. Kafka 替换 RabbitMQ（海量吞吐、消息回溯）
6. 评论楼中楼（parent_id）与分库分表
