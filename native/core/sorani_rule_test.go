package core

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSoraniRuleParse(t *testing.T) {
	ruleJSON := `{
	  "主页url": "https://api.sorani.cc/sorani-cms",
	  "请求头": "User-Agent$Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	  "分类": "TV番剧$1#剧场版$2#特摄剧集$5#国漫$7",
	  "分类url": "https://api.sorani.cc/sorani-cms/api/video?page={catePg}&size=24&categoryId={cateId}",
	  "搜索url": "https://api.sorani.cc/sorani-cms/api/video?keyword={wd}&page={catePg}&size=24",
	  "数组": "j:data.records",
	  "标题": "j:title",
	  "链接": "https://api.sorani.cc/sorani-cms/api/video/+j:id",
	  "列表图片": "j:cover",
	  "副标题": "j:categoryName",
	  "详情url": "https://api.sorani.cc/sorani-cms/api/video/{id}",
	  "影片名称": "j:data.title",
	  "简介": "j:data.summary",
	  "封面": "j:data.cover",
	  "类型": "j:data.categoryName",
	  "导演": "j:data.director",
	  "状态": "j:data.statusText",
	  "播放数组": "j:data.episodes",
	  "播放列表": "},{",
	  "播放标题": "\"title\":\"&&\"",
      "播放链接": "https://api.sorani.cc/sorani-cms/api/video/episode/+\"episodeId\":&&,+/play",
	  "跳转播放链接": "j:data.playUrl"
	}`

	rule, ok := parseXBPQRule(ruleJSON)
	if !ok {
		t.Fatal("规则解析失败")
	}

	// 模拟详情 API 响应
	detailBody := `{"code":200,"data":{"id":4640,"title":"Re:Zero","episodes":[
		{"episodeId":65650,"videoId":4640,"videoTitle":null,"title":"第01集","playUrl":null,"duration":null,"viewCount":0,"episodeLabel":"第01集","episodeOrder":1,"isVip":false,"previewSeconds":0,"enable":true,"createTime":"2026-08-12","updateTime":"2026-08-12","enabled":true,"vipRequired":false,"formattedDuration":"00:00:00","formattedPreviewDuration":"00:00","hlsFormat":false,"mp4Format":false,"parseFormat":false,"enableText":"启用","vipRequiredText":"免费","shortFormattedDuration":"00:00","playFormat":"未知","displayTitle":"第01集 第01集"},
		{"episodeId":65741,"videoId":4640,"videoTitle":null,"title":"第02集","playUrl":null,"duration":null,"viewCount":0,"episodeLabel":"第02集","episodeOrder":2,"isVip":false,"previewSeconds":0,"enable":true,"createTime":"2026-08-19","updateTime":"2026-08-19","enabled":true,"vipRequired":false,"formattedDuration":"00:00:00","formattedPreviewDuration":"00:00","hlsFormat":false,"mp4Format":false,"parseFormat":false,"enableText":"启用","vipRequiredText":"免费","shortFormattedDuration":"00:00","playFormat":"未知","displayTitle":"第02集 第02集"}
	]}}`

	episodes, routes := xbpqEpisodes(detailBody, "https://api.sorani.cc/sorani-cms/api/video/4640", rule)
	t.Logf("解析到 %d 集, %d 条线路", len(episodes), len(routes))

	for i, ep := range episodes {
		t.Logf("集[%d]: title=%q, url=%q", i, ep.title, ep.url)
		if ep.url == "" {
			t.Errorf("集[%d] URL 为空", i)
		}
	}

	// 检查播放链接是否是正确的 URL
	if len(episodes) > 0 {
		firstURL := episodes[0].url
		if !strings.HasPrefix(firstURL, "https://api.sorani.cc/sorani-cms/api/video/episode/") {
			t.Errorf("播放链接前缀错误: %q", firstURL)
		}
		if !strings.HasSuffix(firstURL, "/play") {
			t.Errorf("播放链接后缀错误: %q", firstURL)
		}
		// 检查中间是否有 episodeId
		if !strings.Contains(firstURL, "episode/65650/play") {
			t.Errorf("episodeId 未正确嵌入: %q", firstURL)
		}
	}

	// 测试跳转播放链接
	playBody := `{"code":200,"data":{"episodeId":65650,"canPlay":true,"playUrl":"https://www.sorani-vids.xyz/videos/test/index.m3u8?counts=5&timestamp=123&key=abc","hls":true}}`
	jump := rule.field("跳转播放链接")
	if jump != "j:data.playUrl" {
		t.Errorf("跳转播放链接字段错误: %q", jump)
	}
	candidates := xbpqJumpCandidates(playBody, jump)
	t.Logf("跳转候选: %v", candidates)
	if len(candidates) == 0 {
		t.Error("跳转播放链接未提取到候选")
	}
	if len(candidates) > 0 && !strings.Contains(candidates[0], "index.m3u8") {
		t.Errorf("跳转候选不是 m3u8: %q", candidates[0])
	}

	// 测试链接拼接
	linkPattern := rule.field("播放链接")
	t.Logf("播放链接 pattern: %q", linkPattern)
	
	item := `{"episodeId":65650,"videoId":4640,"title":"第01集"}`
	link := xbpqCutOnce(item, linkPattern)
	t.Logf("截取结果: %q", link)
	if link != "https://api.sorani.cc/sorani-cms/api/video/episode/65650/play" {
		t.Errorf("链接截取错误: 期望 https://api.sorani.cc/sorani-cms/api/video/episode/65650/play, 实际 %q", link)
	}

	// 测试列表数组
	listBody := `{"code":200,"data":{"records":[{"id":4765,"title":"测试","categoryName":"TV番剧","cover":"https://img.sorani.net/cover.webp"}],"total":1,"pages":1}}`
	items := xbpqExtractItems(nil, rule, "https://api.sorani.cc/sorani-cms", "https://api.sorani.cc/sorani-cms/api/video?page=1&size=24&categoryId=1", "")
	t.Logf("列表条目数: %d", len(items))
}
