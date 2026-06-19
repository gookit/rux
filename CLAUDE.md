# For AI

## go 开发规范

- 如果当前任务有进度跟进文件/计划文件，需要在完成后更新 checkbox 为已完成
- 多阶段任务，需要在每个 **子阶段完成** 后更新进度并提交 Git 提交，不要在全部完成后提交一大堆变动。子阶段改动多时按功能点拆分提交，避免一次累计大量变动。
  - commit 标题前缀按 conventional commit（`feat / fix / refactor / docs / chore`）+ scope（如 `feat(decoration): ...`）
- 一般代码自解释，无需额外注释；但是 **关键的** 方法或逻辑点 需要添加注释说明，方便理解。
- 没有明确指定时，如果当前功能改动涉及的逻辑文件 **超过3个** 或者 **超过100行业务代码**，需要向用户确认后再实施
- 启动 go server 项目服务时，先 go build 再启动build后的程序，避免直接 go run 需要每次授权卡住

### 单元测试编写

- 使用 `github.com/gookit/goutil/x/assert` 断言结果（`testutil/assert` 已弃用）
- 同一个方法的多个用例可以使用 `t.Run()` 包裹进行分组

`require` 断言结果的写法：

```go
Require(t, assert.Eq(t, 1, res.ID))

// Standard assertion
assert.Eq(t, expected, actual)
```
