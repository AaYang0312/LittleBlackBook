# xbs — 小红书风格社交 Feed 后端 Demo

面向「高并发读、海量写计数」场景的社交 Feed 后端演示项目（纯后端 API）：推模式 Feed 分发、Redis 实时计数 + MQ 异步落库的最终一致性架构、缓存三问题防护、三层幂等设计。模块化单体，模块边界预留微服务演进。

## 技术栈

| 组件 | 选型 | 备注 |
|---|---|---|
| 语言/框架 | Go + Gin + GORM | 模块化单体 |
| 数据库 | MySQL 8 | 关系数据 + 计数持久化（事实源） |
| 缓存 | Redis 7 | timeline、计数、笔记缓存、判重、空值缓存 |
| 消息队列 | RabbitMQ | 点赞事件、feed 分发事件（at-least-once） |
| 对象存储 | MinIO | 笔记图片，S3 兼容 API |
| ID 生成 | 雪花算法 | 笔记/评论 ID |
| 并发原语 | golang.org/x/sync singleflight | 防缓存击穿 |
| 日志 | log/slog | 结构化 JSON |
| 接口文档 | swaggo/swag | 注解生成 Swagger |
| 部署 | docker-compose | MySQL / Redis / RabbitMQ / MinIO / server / worker |

> **MQ 选型理由**：demo 量级下 RabbitMQ 足够，概念简单、运维轻；Kafka 的优势在海量吞吐与消息回溯，属于演进方向。

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

一个代码库、两个进程（HTTP server + MQ worker）、四个业务模块（user / note / interaction / feed）。每个模块内部 `handler → service → repository` 三层，禁止跨模块 import 其他模块的 repository，需要跨模块数据只能调用其 service——拆微服务时把 service 接口换成 RPC 客户端即可。

## 快速开始

```bash
make up                 # 启动 MySQL / Redis / RabbitMQ / MinIO
make run-server         # 终端 1：HTTP 服务（:8080）
make run-worker         # 终端 2：MQ 消费者（like_event / feed_fanout）
make e2e                # 全链路验收：注册→登录→上传→发笔记→关注→点赞→评论→feed→重建
```

- Swagger 文档：`http://localhost:8080/swagger/index.html`
- 接口前缀：`/api/v1`

## 核心设计决策与面试话术

### Feed 推模式与发现页设计（spec §5.4）

- **关注页**：有关注关系作为集合边界，适合推模式（fan-out on write）——发布笔记时写粉丝的 `feed:inbox:{uid}` ZSet，读时 `ZREVRANGE` 直接取，读路径 O(logN)。
- **发现页**：面向全站笔记池无边界，推模式写放大为全站用户数，不可行。真实方案是推荐系统多路召回（热门/协同过滤/标签）→ 粗排 → 精排，数据源是推荐引擎与 ES 而非收件箱。本 demo 以「全站最新」作为召回源占位，接口契约（游标分页）与推荐场景一致，未来可平滑替换。

### 三层幂等（spec §6.4）

| 层次 | 机制 | 防什么 |
|---|---|---|
| Redis 判重 | `SADD` 返回 0 则不发 MQ | 用户双击、前端重试 |
| MQ 消费 | nack 重试 3 次 → 死信队列 | 消费者宕机丢消息 |
| DB 唯一索引 | `uk_user_note` + `INSERT IGNORE` | MQ at-least-once 重复投递 |

> RabbitMQ 只保证 at-least-once，幂等必须落在消费端与存储层，不能依赖 MQ 不重复。

### 缓存三问题（spec §6.1-6.3）

- **穿透**：回源查不到 → 写空值缓存（TTL 30s），命中直接 404。演进：入口层布隆过滤器。
- **击穿**：回源必须经 `singleflight.Group`，同一 noteId 并发回源合并为一次 DB 查询。
- **雪崩**：TTL 随机抖动（1h ± 10min）；Redis 不可用时读路径降级为直查 MySQL + 日志告警（演进：sentinel/熔断）。

### 计数按操作频率分级（spec §5.5）

- **点赞/收藏（高频写）**：Redis `SADD` 判重 + `INCR` 实时计数 → 发 MQ `like_event` → worker 异步落库，HTTP 接口不等落库完成即返回。
- **评论（低频写）**：不走 MQ，直插 `comments` + 同步更新计数，简化链路。

按操作频率分级处理：高频走异步削峰，低频走同步简化链路。

### 计数一致性与重建（spec §6.5）

- 实时读永远以 Redis 计数为准；MySQL 关系表（likes/collects/comments）是唯一事实源。
- `POST /internal/rebuild-counts`：从关系表 `COUNT(*)` 重建 Redis 计数与 notes 计数列，用于 Redis 数据丢失后的恢复。

## 演进方向

1. **推拉结合**：粉丝数超阈值的作者走拉模式，普通人走推模式，读时合并两路。
2. **发现页接推荐系统/ES 多路召回**，替换「全站最新」占位召回。
3. **布隆过滤器**替代空值缓存；**sentinel/熔断**替代简单降级。
4. **模块边界即微服务边界**：service 接口换 RPC（go-zero/Kratos + gRPC），user/note/interaction/feed 拆为四个服务。
5. **Kafka 替换 RabbitMQ**（海量吞吐、消息回溯）。
6. **评论楼中楼**（parent_id）与**分库分表**。

## 已知取舍（演进项）

| 取舍 | 说明 | 演进 |
|---|---|---|
| feed_fanout 无 DLQ | 分发失败直接丢弃，关注页可能短暂缺失新笔记 | 消费失败进死信队列重放 |
| `/internal/rebuild-counts` 无鉴权 | 内部接口，未加认证 | 内网隔离 / 加鉴权 |
| MinIO bucket 公共读 | 图片直链可访问 | 私有 bucket + 签名 URL |
| `BatchByIDs` 未走缓存批量优化 | 批量回源后逐条回填，未做 pipeline | 批量读缓存 + 批量回填 |
