package gleaner_test

import (
	"strings"
	"testing"

	"github.com/soulteary/gleaner"
)

const goldenHTML = `<!DOCTYPE html><html><head><title>星河科技发布新一代AI芯片：算力提升三倍</title></head><body>
<div class="article" id="artibody">
<p><strong>专题：2026人工智能芯片大会</strong></p>
<p>新浪科技讯 北京时间6月24日消息，星河科技今日正式发布新一代人工智能芯片"星河一号"，该芯片采用先进制程工艺，单卡算力较上一代提升三倍，整体功耗降低约百分之四十。</p>
<p>星河一号专为大模型的训练与推理而设计，单颗芯片即可支持千亿参数模型的高效运行，并配套完善的软件生态，为人工智能应用提供强劲的算力支撑。</p>
<p>星河科技表示，该芯片采用多芯粒互联架构，将多个计算单元通过高速总线连接，在单封装内提供超大规模的并行算力，同时显著优化了能效比。</p>
<p>多位业内人士认为，星河一号的发布标志着国产人工智能芯片在算力与能效方面取得重要突破，有望显著降低大模型的训练成本，推动整个人工智能产业快速发展。</p>
</div>
</body></html>`

func TestGetArticleSync(t *testing.T) {
	client, err := gleaner.NewClient(gleaner.DefaultOptions())
	if err != nil {
		t.Fatalf("client: %v", err)
	}
	defer client.Close()

	article, err := client.GetArticleSync(goldenHTML, 200, nil)
	if err != nil {
		t.Fatal(err)
	}
	if article.Content == "" {
		t.Fatal("empty content")
	}
	if article.Title == "" {
		t.Fatal("empty title")
	}
	if article.Summary == "" {
		t.Fatal("empty summary")
	}
	if !strings.Contains(article.Content, "星河一号") {
		t.Fatalf("content missing keywords: %q", article.Content[:min(80, len(article.Content))])
	}
}

func TestSummarizeAPI(t *testing.T) {
	client, err := gleaner.NewClient(gleaner.DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	summary := client.Summarize(goldenHTML, 120, nil, "星河一号AI芯片")
	if summary == "" {
		t.Fatal("empty summary")
	}
}

func TestGetArticleSyncRatioLength(t *testing.T) {
	client, err := gleaner.NewClient(gleaner.DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	opts := gleaner.DefaultExtractOptions(0.04, nil)
	article, err := client.GetArticleSyncWithOptions(goldenHTML, opts)
	if err != nil {
		t.Fatal(err)
	}
	if article.Summary == "" {
		t.Fatal("empty summary")
	}
}

func TestSentimentAPI(t *testing.T) {
	client, err := gleaner.NewClient(gleaner.DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()

	r := client.Sentiment("非常好看，开心", nil)
	if r.Score == 0 && len(r.Tokens) == 0 {
		t.Fatal("expected sentiment result")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
