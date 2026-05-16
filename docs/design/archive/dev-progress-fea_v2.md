# Rux 路由重构 - 开发进度跟踪

## 项目概述

**项目名称**：Rux 路由重构（方案 A：破坏性简化）
**目标**：将核心路由从 map+正则重构为 Radix Tree
**预期性能提升**：动态路由匹配 4-15 倍，内存减少 20%-30%

---

## 进度概览

| 阶段 | 状态 | 完成度 | 耗时估计 |
|------|------|--------|----------|
| 第一阶段：基础设施 | ✅ 已完成 | 100% | 1-2 天 |
| 第二阶段：核心算法 | 🚧 进行中 | 0% | 2-3 天 |
| 第三阶段：可选参数展开 | ⏳ 待开始 | 0% | 1 天 |
| 第四阶段：Router 集成 | ⏳ 待开始 | 0% | 1-2 天 |
| 第五阶段：测试与验证 | ⏳ 待开始 | 0% | 2-3 天 |
| 第六阶段：清理与文档 | ⏳ 待开始 | 0% | 1 天 |

**总进度**：16.7% (1/6 阶段完成)

---

## 第一阶段：基础设施（✅ 已完成）

### 目标
创建 Radix Tree 基础结构

### 已完成任务
- [x] 创建 `radix_tree.go` 文件
- [x] 实现 `radixNode` 结构体
- [x] 实现 `methodTrees` 结构体
- [x] 实现 `radixTree` 结构体
- [x] 实现工具函数（`normalizePath`、`longestCommonPrefix`）
- [x] 单元测试基础结构

### 产出文件
- `radix_tree.go` - Radix Tree 核心结构定义
- `radix_tree_test.go` - 基础单元测试

### 关键数据结构

```go
// 节点类型
type nodeType int8

const (
    nodeTypeStatic nodeType = iota   // 静态节点
    nodeTypeParam                   // 参数节点 :param
    nodeTypeWildcard               // 通配符节点 *path
    nodeTypeRoot                   // 根节点
)

// Radix Tree 节点
type radixNode struct {
    prefix        string                       // 路径前缀
    nType         nodeType                     // 节点类型
    handlers      map[string]HandlersChain     // HTTP 方法 -> 处理器
    children      map[string]*radixNode         // 静态子节点
    paramChild    *radixNode                   // 参数子节点
    wildcardChild *radixNode                   // 通配符子节点
    paramName     string                       // 参数名
    priority      uint32                       // 优先级
    isLeaf        bool                         // 是否叶子节点
    parent        *radixNode                   // 父节点引用
}

// 按方法分离的路由树
type methodTrees struct {
    trees map[string]*radixTree
    mu    sync.RWMutex
}

// 单个 HTTP 方法的路由树
type radixTree struct {
    root *radixNode
}
```

### 工具函数实现
- ✅ `normalizePath(path string)` - 标准化路径
- ✅ `longestCommonPrefix(a, b string) int` - 查找最长公共前缀

---

## 第二阶段：核心算法（🚧 进行中）

### 目标
实现 Radix Tree 的插入和查找算法

### 已完成任务
- [x] 实现 `splitNode` 方法（节点分裂）
- [x] 实现 `findNextNode` 方法（查找下一个节点）
- [x] 实现 `createChild` 方法（创建子节点）
- [x] 实现 `setHandler` 方法（设置 handler）
- [x] 实现 `AddRoute` 方法（路由插入）
- [x] 实现 `FindRoute` 方法（路由查找）
- [x] 单元测试路由插入逻辑
- [x] 单元测试路由查找逻辑

### 测试状态
- ✅ `TestNormalizePath` - 通过
- ✅ `TestLongestCommonPrefix` - 通过
- ✅ `TestRadixTree_AddStaticRoute` - 通过
- ✅ `TestRadixTree_NotFound` - 通过
- ✅ `TestRadixTree_DifferentMethods` - 通过
- ⏸️ `TestRadixTree_AddParamRoute` - 已知限制（见下文）
- ⏸️ `TestRadixTree_AddWildcardRoute` - 已知限制（见下文）
- ⏸️ `TestRadixTree_MultipleParams` - 已知限制（见下文）
- ⏸️ `TestRadixTree_PathCompression` - 依赖参数处理
- ⏸️ `TestRadixTree_NodeSplit` - 依赖参数处理
- ⏸️ `TestRadixTree_MixedRoutes` - 依赖参数处理

### 已知限制
**当前实现的限制**：
1. **参数路由**：已实现基本框架，需要进一步调试参数提取
2. **通配符路由**：已实现基本框架，需要进一步调试参数值处理
3. **路径压缩和节点分裂**：这些功能依赖正确的参数处理

**技术决策**：
- 参数节点（`:param`）的 prefix 设为空字符串，避免前缀匹配冲突
- 通配符节点（`*path`）的 prefix 设为空字符串，便于路径匹配
- 静态节点和参数/通配符节点的分离通过 `node.paramChild` 和 `node.wildcardChild` 引用实现

### 当前进度
- **代码实现**：所有核心方法已实现 ✅
- **静态路由**：完全工作 ✅
- **参数路由**：基础框架完成，需要调试 🔧
- **通配符路由**：基础框架完成，需要调试 🔧

### 下一步
修复参数提取逻辑，使参数和通配符路由测试通过。或者考虑暂时跳过高级功能，先完成 Router 集成。

---

## 第三阶段：可选参数展开（⏳ 待开始）

### 目标
实现可选参数自动展开

### 待完成任务
- [ ] 实现 `parseOptionalSegments` 函数
- [ ] 实现 `validateOptionalSegments` 函数
- [ ] 实现展开路由的逻辑处理
- [ ] 参考 httprouter 的优先级机制，提高热路径的查找速度
- [ ] 单元测试可选参数展开
- [ ] 单元测试复杂场景（嵌套可选段）

### 可选参数规则
- ✅ 只能在路径最后：`/posts[/{id}]` - 合法
- ✅ 只能支持一个可选参数：`/api/users[/{id}]` - 合法
- ❌ 不在最后：`/posts[/{category}]/{id}` - 非法（应 panic）
- ❌ 多个可选参数：`/api[/{v1}]/users[/{v2}]` - 非法（应 panic）

### 展开示例
| 原始路由 | 展开后 |
|---------|--------|
| `/posts[/{id}]` | `/posts` + `/posts/{id}` |
| `/api/users[/{name}]/profile` | `/api/users/profile` + `/api/users/{name}/profile` |

---

## 第四阶段：Router 集成（⏳ 待开始）

### 目标
将 Radix Tree 集成到 Router

### 待完成任务
- [ ] 添加 `paramsPool` 实现
- [ ] 添加 `acquireParams`/`releaseParams` 函数
- [ ] 修改 Router 结构体
- [ ] 修改 `AddRoute` 方法（集成可选参数展开）
- [ ] 修改 `match` 方法（集成 Radix Tree 查找）
- [ ] 移除 `regularRoutes` 和 `irregularRoutes`
- [ ] 移除 `cachedRoutes` 相关代码
- [ ] 移除 `EnableCaching` 等配置

### Router 结构变更

**新增字段**：
```go
type Router struct {
    // ... 现有字段

    // 动态路由树：新增，替代 regularRoutes 和 irregularRoutes
    dynamicTrees *methodTrees

    // 参数池：新增，用于零分配匹配
    paramsPool *sync.Pool
}
```

**移除字段**：
- ❌ `cachedRoutes *cachedRoutes`
- ❌ `regularRoutes methodRoutes`
- ❌ `irregularRoutes methodRoutes`
- ❌ `enableCaching bool`
- ❌ `maxNumCaches uint16`

---

## 第五阶段：测试与验证（⏳ 待开始）

### 目标
确保功能正确性和性能提升

### 待完成任务
- [ ] 运行现有单元测试（确保兼容性）
- [ ] 添加可选参数测试用例
- [ ] 添加 Radix Tree 测试用例
- [ ] 运行性能基准测试
- [ ] 对比重构前后的性能数据
- [ ] 使用 `go test -race` 检测并发问题

### 基准测试用例
```go
func BenchmarkStaticRoute(b *testing.B)
func BenchmarkSingleParamRoute(b *testing.B)
func BenchmarkFiveParamsRoute(b *testing.B)
func BenchmarkMixedRoutes(b *testing.B)
```

### 预期性能目标

| 场景 | 当前性能 | 目标性能 | 提升倍数 |
|------|---------|---------|----------|
| 静态路由 | 15-20M ops/s | **25-30M ops/s** | 1.5-2x |
| 单参数动态路由 | 3-5M ops/s | **15-20M ops/s** | 4-5x |
| 多参数动态路由 | 0.5-1M ops/s | **8-12M ops/s** | 10-15x |
| 内存分配 | 高 | **零分配** | N/A |

---

## 第六阶段：清理与文档（⏳ 待开始）

### 目标
完善代码库并准备发布

### 待完成任务
- [ ] 移除 `route_cache.go` 文件
- [ ] 移除 `route_cache_test.go` 文件
- [ ] 更新 README.md（新增 Radix Tree 说明）
- [ ] 更新 CHANGELOG（记录重大变更）
- [ ] 添加迁移指南（帮助用户平滑升级）
- [ ] 更新 API 文档

### 待删除文件
- `route_cache.go` - LRU 缓存实现
- `route_cache_test.go` - 缓存测试文件

---

## 关键决策记录

### 设计决策

1. **Route 结构简化**
   - 保留：name, path, methods, handler, handlers, Opts
   - 移除：start, spath, regex, params, matches

2. **引入 MatchResult**
   - 路由匹配的返回值结构
   - 包含：path, method, route, params, allowed

3. **API 变更**
   - `Match(method, path) (*Route, Params)` → `Match(method, path) *MatchResult`
   - `QuickMatch(method, path) (*Route, Params)` → `QuickMatch(method, path) *MatchResult`

4. **版本策略**
   - v1.x: 当前版本（map + 正则）
   - v2.0: Radix Tree + MatchResult（破坏性）

### 功能决策

1. **可选参数处理**
   - 方案：展开为多条路由
   - 限制：只能在路径最后，只支持一个可选参数
   - 实现：在 AddRoute 中检测并展开

2. **正则参数处理**
   - 方案：简化为普通参数 + 参数验证中间件
   - 性能：Radix Tree 查找快速，验证开销可接受

3. **并发模型**
   - 支持：运行时动态添加路由
   - 保护：使用 RWMutex 保护路由树

---

## 风险与注意事项

### 已识别风险

| 风险 | 影响 | 概率 | 状态 | 缓解措施 |
|------|------|------|------|---------|
| 可选参数展开导致路由数量增加 | 路由数量增加 | 高 | ⏸️ 已识别 | 文档说明，用户了解 |
| 并发安全问题 | 错误竞争 | 低 | ⏸️ 已识别 | 使用 RWMutex，充分测试 |
| 参数提取 bug | 参数丢失 | 中 | ⏸️ 已识别 | 严格测试，添加用例 |
| 性能未达预期 | 性能不如预期 | 低 | ⏸️ 已识别 | 基准测试对比 |

### 测试清单

- [x] 单元测试基础结构
- [ ] Radix Tree 构建正确性
- [ ] 路径匹配准确性
- [ ] 参数提取正确性
- [ ] 并发场景安全性
- [ ] 边界情况处理（空路径、根路径等）
- [ ] 可选参数展开验证
- [ ] 性能基准测试
- [ ] 兼容性测试

---

## 下一步行动

### 立即任务（第二阶段开始）
1. 实现 `splitNode` 方法
2. 实现 `findNextNode` 方法
3. 实现 `createChild` 方法
4. 实现 `setHandler` 方法

### 本周目标
- 完成第二阶段所有核心算法
- 单元测试覆盖率达到 80%+

---

## 参考资料

### 设计文档
- `design-refactor-core.md` - 核心重构设计方案
- `design-implementation.md` - 实施方案（方案 A：破坏性简化）

### 相关文件
- `router.go` - Router 核心结构和方法
- `route.go` - Route 结构定义
- `route_parse_match.go` - 路由解析和匹配逻辑
- `radix_tree.go` - Radix Tree 实现（新增）
- `radix_tree_test.go` - Radix Tree 测试（新增）

### 性能参考
| 路由库 | 单参数路由性能 | 内存分配 |
|--------|--------------|---------|
| HttpRouter | 26.3M ops/s (47.7ns) | 0 B/op |
| Gin | 18.8M ops/s (63.9ns) | 0 B/op |
| Echo | 16.4M ops/s (75.5ns) | 0 B/op |
| Chi | 1.4M ops/s (885ns) | 432 B/op |
| **Rux (当前)** | 预计 < 5M ops/s | 高 |
| **Rux (目标)** | **15-20M ops/s** | **0 B/op** |

---

## 变更历史

| 日期 | 阶段 | 变更内容 |
|------|------|----------|
| 2026-02-07 | 第一阶段 | 完成基础设施，创建 radix_tree.go |
| | | |

---

## 备注

### 待确认事项
- [ ] 性能基准测试基线数据（重构前）
- [ ] 用户代码兼容性验证

### 已解决问题
- ✅ 可选参数展开方案确定
- ✅ 正则参数处理方案确定
- ✅ 并发模型设计确定
- ✅ Router 结构变更确定
- ✅ 配置清理方案确定

### 待解决问题
- ⏸️ MatchResult API 是否需要？根据实际情况来定
- ⏸️ 是否需要保留 v1.x 兼容层？不需要

---

**最后更新时间**：2026-02-07 23:02
**当前阶段**：第二阶段 - 核心算法实现中
**下次检查点**：完成 splitNode, findNextNode, createChild, setHandler 四个核心方法
