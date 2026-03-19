package crawler

import (
	"fmt"
	"github.com/chromedp/cdproto/browser"
	"github.com/chromedp/chromedp"
	"github.com/saxon134/go-utils/saData"
	"github.com/saxon134/go-utils/saLog"
	"os"
	"testing"
)

func TestStep(*testing.T) {
	var steps = make([]Step, 0, 10)
	_ = saData.StrToModel(`
		[{
			"act": "navigate",
			"val": "https://www.baidu.com/"
		}, {
			"act": "sleep",
			"ms": 2000
		}, {
			"act": "fill",
			"sel": "#chat-textarea",
			"val": "今天天气咋样"
		}, {
			"act": "click",
			"sel": "#chat-submit-button"
		}, {
			"act": "sleep",
			"ms": 1000
		}, {
			"act": "sleep",
			"ms": 5000
		}]
	`, &steps)

	//启动浏览器
	var ctx = NewContextWithPort(
		9013,
		chromedp.UserDataDir("./.user-data-crawler"),
	)

	//设置默认下载路径，需提前创建好目录
	_ = os.MkdirAll("./.download-crawler", os.ModePerm)
	var err = chromedp.Run(ctx, browser.SetDownloadBehavior(browser.SetDownloadBehaviorBehaviorAllowAndName).WithDownloadPath("./.download-crawler").WithEventsEnabled(true))
	if err != nil {
		saLog.Err(err)
	}

	for _, step := range steps {
		_ = step.Run(ctx)
	}
	fmt.Println("完成任务")
}
