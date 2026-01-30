# Simple Go RAG Assistant 🚀

这是一个基于 Go 语言实现的轻量级 RAG（检索增强生成）原型。

## 🌟 项目亮点
- **高性能并发：** 利用 Go 协程（Goroutine）模拟多模型并行调用。
- **本地知识库：** 支持从 `data.txt` 实时检索上下文。
- **AI 集成：** 接入 DeepSeek/OpenAI 兼容接口进行智能回答润色。

## 🛠️ 技术栈
- **Language:** Go 1.24+
- **Protocol:** OpenAI API Protocol / MCP 思想
- **Library:** Standard Library (net/http, bufio, encoding/json)

## 🏃 快速启动
1. 在 `data.txt` 中添加你的本地知识。
2. 在 `main.go` 中配置你的 `API_KEY`。
3. 运行：
   ```bash
   go run main.go