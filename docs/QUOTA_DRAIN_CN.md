# Quota Drain 路由规则

## 目标

Quota Drain 的目标是减少付费订阅额度在重置时浪费，同时保留现有 priority、cooldown、模型匹配和 session affinity 语义。

## 选择顺序

1. 先排除 disabled、cooldown 或相关额度已经耗尽的账号。
2. priority 数值越大越先处理。
3. 只在同一 priority 层内比较额度。
4. 长周期窗口按以下紧迫度排序：

~~~
紧迫度 = 剩余额度百分比 / 距离重置剩余小时数
~~~

数值越大越优先。

例：

~~~
账号 A：剩余 80%，8 小时后重置  => 10
账号 B：剩余 30%，24 小时后重置 => 1.25
~~~

优先使用 A。

## 长短周期

周期小于 24 小时的窗口不参与长期排序，只负责可用性门控。

因此 5 小时额度还剩 1% 时账号仍可被长期规则选中；一旦相关 5 小时窗口耗尽，该账号会被跳过，直到额度恢复。

24 小时及以上窗口参与紧迫度排序。

## Provider 范围

当前只主动采集 OAuth 类型的：

- Antigravity
- Codex
- xAI

Antigravity：
Gemini 配额池与 Claude/GPT 配额池分开匹配。

Codex：
通用 rate limit 对所有 Codex 模型生效；additional rate limits 只匹配对应 ScopeModel。

xAI：
Weekly 总额度参与；GrokBuild 映射普通文本请求，GrokImagine / image 映射图片请求。当前 GrokChat 产品池不参与路由排序。

## 刷新和过期

~~~
刷新周期：5 分钟
数据过期：10 分钟
并发采集 worker：最多 4
~~~

当同一 provider 同一 priority 层中存在缺失、过期或无法形成长期紧迫度的数据时，策略安全退回稳定轮询，避免未知账号被永久饿死。

## Pin 行为

Quota Drain 选中账号后会尽量继续使用该账号，而不是每个请求重新切换，以减少 Prompt / KV Cache 抖动。

以下情况会触发重新选择：

- 已固定账号耗尽或不可用。
- cooldown 使账号不可选。
- 候选账号集合变化。
- 长周期 reset 时间发生明显滚动。
- 新的长期额度周期开始。

## priority

未设置 priority 时默认值为 0。希望账号完全按额度紧迫度调度时，同一类账号保持相同 priority 即可。

## 已知限制

- Antigravity 分组和周期识别依赖上游配额响应中的可识别标签；上游字段或文案改变时可能退回普通轮询。
- Codex additional_rate_limits 的模型范围依赖上游返回的 limit_name / metered_feature。
- xAI 当前只将 GrokBuild 和 GrokImagine 作为产品级路由范围。
- 未经过实际账号配额接口验证的新上游格式，应先在测试环境观察再扩大使用。
