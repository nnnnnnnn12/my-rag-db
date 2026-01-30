package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

type Config struct {
	Synonyms map[string][]string `json:"synonyms"`
}

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

func loadConfig(fileName string) (*Config, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var config Config
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

// 加载文件夹下的所有文本文件内容
func loadAllDocs(folderPath string) ([]string, error) {
	var allDocs []string
	// 读取文件夹下的所有文件
	files, err := os.ReadDir(folderPath)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		// 只读取以 .txt 结尾的文件
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".txt") {
			filePath := folderPath + "/" + file.Name()
			content, err := os.ReadFile(filePath)
			if err != nil {
				fmt.Printf("读取文件 %s 失败: %v\n", file.Name(), err)
				continue
			}
			// 将内容按行拆分并加入知识库
			lines := strings.Split(string(content), "\n")
			for _, line := range lines {
				if strings.TrimSpace(line) != "" {
					allDocs = append(allDocs, line)
				}
			}
		}
	}
	return allDocs, nil
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

// 模拟语义打分：根据关键词和预设的语义关系计算匹配度
func calculateScore(doc, query string, config *Config) float64 {
	score := 0.0
	doc = strings.ToLower(doc)
	query = strings.ToLower(query)

	// 基础匹配：如果直接包含关键词，给最高分
	if strings.Contains(doc, query) {
		score += 10.0
	}

	// --- 也就是你刚才问的那段代码，添加在这里 ---
	for key, words := range config.Synonyms {
		// 只要 query 包含 key（例如“冷”），或者包含 words 里的任何一个（例如“气温”），就视为命中
		match := strings.Contains(query, key)
		if !match {
			for _, w := range words {
				if strings.Contains(query, w) {
					match = true
					break
				}
			}
		}

		// 如果用户提问命中了语义词，且文档(doc)里含有核心词(key)，则加分
		if match && strings.Contains(doc, key) {
			score += 5.0
		}
	}
	return score
}

// --- 主逻辑 ---
func main() {
	// 1. 初始化配置和知识库
	config, _ := loadConfig("config.json")
	knowledgeBase, _ := loadAllDocs("docs")
	apiKey := "sk-54856bff18774119952f437b26705f82" // 别忘了填入你的 Key

	// 2. 创建一个默认的 Gin 引擎
	r := gin.Default()
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Next()
	})
	// 3. 定义一个 GET 接口：/chat
	r.GET("/chat", func(c *gin.Context) {
		// 从网址参数里获取问题，例如 /chat?q=气温
		query := c.Query("q")
		if query == "" {
			c.JSON(400, gin.H{"error": "请提供问题关键词 q"})
			return
		}

		// --- 下面就是你刚才写的并发检索逻辑 ---
		type SearchResult struct {
			Score   float64
			Context string
		}
		resultChan := make(chan SearchResult, len(knowledgeBase))
		var wg sync.WaitGroup

		for _, doc := range knowledgeBase {
			wg.Add(1)
			go func(d string) {
				defer wg.Done()
				score := calculateScore(d, query, config)
				if score > 0 {
					resultChan <- SearchResult{Score: score, Context: d}
				}
			}(doc)
		}

		go func() {
			wg.Wait()
			close(resultChan)
		}()

		var bestContext string
		var maxScore float64
		for res := range resultChan {
			if res.Score > maxScore {
				maxScore = res.Score
				bestContext = res.Context
			}
		}

		// --- 调用 AI 生成回答 ---
		finalPrompt := fmt.Sprintf("背景资料：%s\n用户问题：%s", bestContext, query)
		answer := askAI(apiKey, finalPrompt)

		// --- 以 JSON 格式把结果返回给浏览器 ---
		c.JSON(200, gin.H{
			"query":    query,
			"context":  bestContext,
			"score":    maxScore,
			"ai_reply": answer,
		})
	})

	// 4. 启动 Web 服务，默认监听 8080 端口
	fmt.Println("🚀 RAG 机器人 Web 服务已启动：http://localhost:8080/chat?q=你的问题")
	r.Run(":8080")
}
