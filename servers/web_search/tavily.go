package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/daodao97/xgo/xjson"
	"github.com/daodao97/xgo/xrequest"
)

var TavilySearchURL = "https://api.tavily.com/search"
var TavilySearchAPIKey = os.Getenv("TAVILY_SEARCH_API_KEY")

type TavilySearchReq struct {
	Query string `json:"query"`
}

type TavilySearchResult struct {
	Title   string  `json:"title"`
	URL     string  `json:"url"`
	Content string  `json:"content"`
	Score   float64 `json:"score"`
}

type TavilySearchResp struct {
	Results []TavilySearchResult `json:"results"`
}

func TavilySearch(req *TavilySearchReq) (*TavilySearchResp, error) {
	if req.Query == "" {
		return nil, fmt.Errorf("搜索查询词不能为空")
	}

	resp, err := xrequest.New().
		SetHeader("Content-Type", "application/json").
		SetBody(map[string]interface{}{
			"api_key":      TavilySearchAPIKey,
			"query":        req.Query,
			"search_depth": "basic",
			"max_results":  10,
		}).
		Post(TavilySearchURL)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("call tavily search error %s", resp.String())
	}

	results := []TavilySearchResult{}
	_results := resp.Json().Get("results").Array()

	for _, result := range _results {
		_result := xjson.New(result)
		results = append(results, TavilySearchResult{
			Title:   _result.Get("title").String(),
			URL:     _result.Get("url").String(),
			Content: _result.Get("content").String(),
			Score:   _result.Get("score").Float(),
		})
	}

	return &TavilySearchResp{
		Results: results,
	}, nil
}
