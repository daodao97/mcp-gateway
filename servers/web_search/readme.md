# web_search 项目文档

## 简介

web_search 是一个 MCP SSE 协议的网络搜索服务，允许用户通过简单的接口执行网络查询并获取格式化的搜索结果。该服务使用 Tavily 搜索 API 来获取高质量的网络搜索结果，并以 Markdown 格式返回结果，便于在各种平台上展示。

## 功能特点

- 通过 MCP 协议提供网络搜索工具
- 使用 SSE (Server-Sent Events) 传输方式实现实时数据流
- 自动注册到 MCP 网关，便于集成到更大的服务生态系统
- 返回格式化的 Markdown 搜索结果，包含标题、URL 和内容摘要
- 支持 Docker 容器化部署

## 安装与部署

### 环境要求

- Go 1.24+
- Tavily API 密钥

### 直接运行

```bash
# 设置 Tavily API 密钥
export TAVILY_SEARCH_API_KEY="your_api_key_here"

# 可选：设置 MCP 服务器端口
export MCP_SERVER_PORT="8080"

# 可选：设置 MCP 网关域名
export MCP_GATEWAY_DOMAIN="http://localhost:3121"

# 运行服务
go run main.go
```

### Docker 部署

```bash
# 构建 Docker 镜像
docker build -t web_search .

# 运行容器
docker run -p 8080:8080 \
  -e TAVILY_SEARCH_API_KEY="your_api_key_here" \
  -e MCP_GATEWAY_DOMAIN="http://localhost:3121" \
  web_search
```

## 环境变量配置

| 环境变量 | 默认值 | 说明 |
|----------|--------|------|
| TAVILY_SEARCH_API_KEY | 无 | Tavily 搜索 API 密钥，必须设置 |
| MCP_SERVER_PORT | 8080 | 服务监听端口 |
| MCP_GATEWAY_DOMAIN | http://localhost:3121 | MCP 网关服务地址 |

## API 使用

### MCP 工具说明

服务提供了一个名为 `web_search` 的 MCP 工具，接受以下参数：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| query | string | 是 | 要搜索的查询内容 |

### 返回格式

服务返回的搜索结果以 Markdown 格式组织，每个结果包含：
- 标题（二级标题格式）
- URL（链接格式）
- 内容摘要（引用块格式）

多个结果之间使用水平分割线分隔。

## 集成到其他服务

web_search 服务会自动向指定的 MCP 网关注册自己，只需确保：

1. MCP 网关已启动并可访问
2. 已正确配置 `MCP_GATEWAY_DOMAIN` 环境变量

注册成功后，其他服务可以通过 MCP 网关调用 web_search 工具。

## 示例

通过发送以下请求到 MCP 网关，可以执行网络搜索：

```json
{
  "tool": "web_search",
  "params": {
    "arguments": {
      "query": "最新人工智能技术发展"
    }
  }
}
```

返回结果示例：

```markdown
### 2024年人工智能发展的10大趋势

[https://www.example.com/ai-trends-2024](https://www.example.com/ai-trends-2024)

> 本文探讨了2024年人工智能发展的十大趋势，包括生成式AI的广泛应用、多模态模型的进步以及AI在医疗健康领域的突破性应用...

---

### 人工智能最新研究进展报告

[https://www.example.com/ai-research-2024](https://www.example.com/ai-research-2024)

> 2024年第一季度，人工智能研究领域取得了多项重大突破，特别是在小样本学习和自监督学习方面...
```

