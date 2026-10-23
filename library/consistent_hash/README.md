# consistent_hash

`library/consistent_hash` 提供了一个面向服务节点变更的一致性任务分配实现。

当前包的核心流程：

- `Init(listener)` 初始化 `ServerCstHash`，创建 `hash_circle.NewSortHash()` 并启动后台监听协程。
- 监听器收到服务节点变化后，计算新增/删除节点并更新哈希环。
- 业务侧通过 `OnChange(key, jobs)` 注册关注的任务集合。
- 节点变化后，系统重新计算当前实例 `env.GetId()` 应负责的 `jobs`，并通过 `chNotice` 通知业务方。

## Flowchart

```mermaid
flowchart TD
    A["调用 Init(listener)"] --> B[创建 ServerCstHash]
    B --> C["创建哈希环 c = hash_circle.NewSortHash()"]
    C --> D["启动后台协程 change()"]

    D --> E["监听 listener.OnChanged()"]
    E --> F{收到变更事件?}

    F -- 是 --> G["onChange(notice)"]
    F -- 否，定时器到期 --> H["listener.OnNow()"]
    H --> G

    G --> I{notice.Err != nil?}
    I -- 是 --> J[记录错误并返回]
    I -- 否 --> K["serverHasDiff(services)"]

    K --> L{add/del 都为空?}
    L -- 是 --> M["服务无变化，返回"]
    L -- 否 --> N["setService(add, del)"]

    N --> N1["新增节点: c.Add(node)"]
    N --> N2["删除节点: c.Remove(node)"]
    N1 --> O[更新 serversMap]
    N2 --> O

    O --> P[遍历 notice 中所有业务 keyInfo]
    P --> Q["noticeJob(keyInfo)"]
    Q --> R["getCurJob(jobs)"]
    R --> S["c.Gets(env.GetId(), jobs)"]
    S --> T[把当前节点应执行的 Jobs 写入 chNotice]

    U["业务调用 OnChange(key, jobs)"] --> V["add(key, jobs)"]
    V --> W{key 已存在?}
    W -- 是 --> X[返回 false / nil]
    W -- 否 --> Y[保存 keyInfo 到 notice]
    Y --> Q

    subgraph HashCircle[hash_circle.SortConsistentHash]
        HC1["Add(node)"] --> HC2["计算 hashSum(node)"]
        HC2 --> HC3[加入 hashCircle]
        HC3 --> HC4[按 NodeHash 排序]
        HC4 --> HC5[记录到 mp]
        HC5 --> HC6[totalHash += nodeHash]

        HD1["Remove(node)"] --> HD2["计算 hashSum(node)"]
        HD2 --> HD3[定位节点索引并删除]
        HD3 --> HD4[从 mp 删除]
        HD4 --> HD5[totalHash -= nodeHash]

        HG1["Gets(targetNode, arr)"] --> HG2{targetNode 在 mp 中?}
        HG2 -- 否 --> HG3[返回 nil]
        HG2 -- 是 --> HG4[找到 targetNode 在环上的索引 idx]
        HG4 --> HG5[遍历任务数组 arr]
        HG5 --> HG6["计算 nodeIdx = (i + totalHash) % len(hashCircle)"]
        HG6 --> HG7{idx == nodeIdx?}
        HG7 -- 是 --> HG8[该任务归当前节点]
        HG7 -- 否 --> HG9[跳过]
    end
```
