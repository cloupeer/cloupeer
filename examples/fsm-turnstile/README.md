# `looplab/fsm` 库示例 (十字转门状态机)

本项目用于演示 `github.com/looplab/fsm` 库 的核心用法，完全独立于 Autopeer 平台。

我们模拟一个地铁十字转门，它有两个状态和两个事件。

## 核心概念

这个库的设计哲学是“显式优于隐式”。它是一个“库”(Library)而不是一个“框架”(Framework)。

1.  **`fsm.NewFSM(initial, events, callbacks)`**
    * 这是唯一的构造函数。你必须在创建时就传入所有的“状态 (States)”、“事件 (Events)”和“回调 (Callbacks)”。

2.  **`fsm.Events`**
    * 一个 `[]EventDesc` 切片。
    * 每个 `EventDesc` 通过 `Name` (事件名), `Src` (源状态列表) 和 `Dst` (目标状态) 来定义一个合法的状态转换。

3.  **`fsm.Callbacks`**
    * 一个 `map[string]Callback`。这是实现“动作 (Actions)”和“守卫 (Guards)”的地方。
    * `fsm` 通过命名约定来触发回调：
        * `before_<EVENT>`: 在事件*之前*触发（**用于实现守卫**）。
        * `leave_<STATE>`: 在离开状态*之前*触发（也可用于守卫）。
        * `enter_<STATE>`: 在进入状态*之后*触发（**用于实现动作**）。
        * `after_<EVENT>`: 在事件*之后*触发（也用于动作）。
        * `enter_state`: 进入*任何*状态后都会触发。

4.  **`fsm.Event(ctx, event, ...args)`**
    * 触发一个事件（状态转换）。`args` 可以向回调函数传递参数。

5.  **`fsm.Current()`**
    * **核心的“显式”特性**：`fsm` *不会*自动修改你的外部对象。你必须*手动*调用 `fsm.Current()` 来获取 FSM 实例的新状态，然后*手动*将其同步回你的业务对象。

## 演示的关键特性

在 `main.go` 中，你将看到：

* **状态**：`locked` (锁定), `unlocked` (解锁)
* **事件**：`coin` (投币), `push` (推门)
* **动作 (Action)**：使用 `enter_unlocked` 和 `enter_locked` 回调来打印闸机动作。
* **守卫 (Guard)**：使用 `before_push` 回调，检查 FSM 是否处于 `locked` 状态。如果是，就调用 `e.Cancel()` 来阻止 `push` 事件发生，并返回一个自定义错误。
* **参数传递**：`coin` 事件将携带一个 `coinType` (string) 参数，并在回调中打印它。

## 如何运行

1.  确保你已安装 `fsm` 库：
    ```bash
    go get [github.com/looplab/fsm](https://github.com/looplab/fsm)
    ```

2.  运行本示例：
    ```bash
    go run examples/fsm-turnstile/main.go
    ```

## 预期输出

```text
[FSM] Initial state: locked
----------------------------------------------------
[Event] ==> Pushing gate...
[Guard] Denied: cannot push a locked gate.
[FSM] State unchanged: locked. Event failed: event push inappropriate in current state locked
----------------------------------------------------
[Event] ==> Inserting a "Quarter"...
[Action] Gate is now unlocked! Please proceed.
[FSM] State changed: unlocked
----------------------------------------------------
[Event] ==> Inserting a "Token"...
[FSM] State unchanged: unlocked. Event failed: event coin inappropriate in current state unlocked
----------------------------------------------------
[Event] ==> Pushing gate...
[Action] Gate is now locked.
[FSM] State changed: locked
```
