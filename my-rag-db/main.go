package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// --- AI API 相关结构体 ---
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

type ChatResponse struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
}

// 调用 AI 的函数
func askAI(apiKey, prompt string) string {
	apiUrl := "https://api.deepseek.com/chat/completions"
	reqBody := ChatRequest{
		Model: "deepseek-chat",
		Messages: []Message{
			{Role: "user", Content: prompt},
		},
	}
	jsonData, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", apiUrl, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "网络错误: " + err.Error()
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var chatResp ChatResponse
	json.Unmarshal(body, &chatResp)

	if len(chatResp.Choices) > 0 {
		return chatResp.Choices[0].Message.Content
	}
	return "AI 没能给出回复"
}

// --- 主逻辑 ---
func main() {
	apiKey := "sk-7fc194096e114465a32221fe902c4ea0" // 替换为真实的 Key

	// 1. 加载本地知识库 (data.txt)
	file, _ := os.Open("data.txt")
	defer file.Close()
	var knowledgeBase []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		knowledgeBase = append(knowledgeBase, scanner.Text())
	}

	// 2. 获取用户提问
	var query string
	fmt.Print("请输入您想咨询的问题关键词: ")
	fmt.Scanln(&query)

	// 3. 检索最相关的上下文
	var context string
	for _, doc := range knowledgeBase {
		if strings.Contains(strings.ToLower(doc), strings.ToLower(query)) {
			context += doc + "\n"
		}
	}

	if context == "" {
		fmt.Println("⚠️ 本地未检索到相关内容，将直接询问 AI...")
		context = "无相关本地背景知识。"
	}

	// 4. 构造 RAG 专属 Prompt
	// 这是 RAG 的核心：告诉 AI，根据我给你的背景资料来回答
	finalPrompt := fmt.Sprintf(`你是我的私人助理。
背景资料：
"""
%s
"""
用户问题：%s
请结合背景资料，用亲切的语气回答用户。`, context, query)

	fmt.Println("\n>>> 正在检索并请求 AI 生成回答...")

	// 5. 获取 AI 回复
	answer := askAI(apiKey, finalPrompt)

	fmt.Println("\n--------------------------------")
	fmt.Println("AI 助手的回答：")
	fmt.Println(answer)
	fmt.Println("--------------------------------")
}
