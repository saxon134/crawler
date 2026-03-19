package crawl

import (
	"context"
	"fmt"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"
	"github.com/saxon134/go-utils/saData"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"
)

type Ctx struct {
	context.Context
	Cancel context.CancelFunc

	Headers map[string]interface{}
	Cookie  string
	Token   string
	InitOk  bool
}

// 指定port开启浏览器进程
/*opts: 通常会设置以下参数：
	chromedp.ExecPath(chromePath),
    chromedp.UserAgent("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36"),
	chromedp.UserDataDir()  - 指定用户空间，且端口号没变化时才能保持登录状态
*/
func NewContextWithPort(port int, opts ...chromedp.ExecAllocatorOption) *Ctx {
	var options = append(
		chromedp.DefaultExecAllocatorOptions[:],
		chromedp.NoDefaultBrowserCheck,
		chromedp.Flag("headless", false),
		chromedp.Flag("blink-settings", "imagesEnabled=true"),
		chromedp.Flag("ignore-certificate-errors", false),
		chromedp.Flag("disable-web-security", false),
		chromedp.Flag("disable-extensions", true),
		chromedp.Flag("disable-plugins", true),
		chromedp.Flag("enable-automation", false),
		chromedp.Flag("accept-language", `zh-CN,zh;q=0.9`),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.Flag("disable-session-crashed-bubble", true), // 关键：禁用崩溃提示
		chromedp.Flag("restore-last-session", true),           // 关键：恢复上次会话
		chromedp.WindowSize(1920, 1080),
	)
	if port > 0 {
		options = append(options, chromedp.Flag("remote-debugging-port", saData.String(port)))
		options = append(options, chromedp.Flag("port", saData.String(port)))
	}

	//自定义参数，会覆盖前面的
	options = append(options, opts...)

	//新建上下文（窗口）
	var ctx, _ = chromedp.NewExecAllocator(context.Background(), options...)
	ctx, _ = chromedp.NewContext(
		ctx,
		chromedp.WithLogf(log.Printf),
	)

	//获取tab页
	var targets, err = chromedp.Targets(ctx)
	if err != nil || len(targets) == 0 {
		return nil
	}

	//连接上第一个tab页面
	var targetID = targets[0].TargetID
	for _, v := range targets {
		if v.Type == "page" {
			targetID = v.TargetID
			break
		}
	}
	var c2, f2 = chromedp.NewContext(ctx, chromedp.WithTargetID(targetID))
	var c = chromedp.FromContext(c2)
	err = target.ActivateTarget(targetID).Do(cdp.WithExecutor(c2, c.Browser))
	if err != nil {
		fmt.Println(err)
	}

	//关闭其他tab页面
	if len(targets) > 1 {
		for _, v := range targets {
			if v.TargetID != targetID && v.Type == "page" {
				err = target.CloseTarget(v.TargetID).Do(cdp.WithExecutor(c2, c.Browser))
				if err != nil {
					fmt.Println(err)
				}
				time.Sleep(time.Millisecond * 200)
			}
		}
	}

	return &Ctx{
		Context: c2,
		Cancel:  f2,
	}
}

// 连接远程浏览器，并连接上第一个tab页
func ConnRemoteContext(ip string, port int) *Ctx {
	var mctx, _ = chromedp.NewRemoteAllocator(context.Background(), fmt.Sprintf("ws://%s:%d/", ip, port))
	if mctx == nil {
		return nil
	}

	//创建上线文
	time.Sleep(time.Millisecond * 500)
	var ctx, _ = chromedp.NewContext(mctx, chromedp.WithLogf(log.Printf))
	time.Sleep(time.Second)

	//获取tab页
	var targets, err = chromedp.Targets(ctx)
	if err != nil || len(targets) == 0 {
		return nil
	}

	//连接上第一个tab页面
	var targetID = targets[0].TargetID
	for _, v := range targets {
		if v.Type == "page" {
			targetID = v.TargetID
			break
		}
	}
	var c2, f2 = chromedp.NewContext(ctx, chromedp.WithTargetID(targetID))
	var c = chromedp.FromContext(c2)
	err = target.ActivateTarget(targetID).Do(cdp.WithExecutor(c2, c.Browser))
	if err != nil {
		fmt.Println(err)
	}

	//关闭其他tab页面
	if len(targets) > 1 {
		for _, v := range targets {
			if v.TargetID != targetID && v.Type == "page" {
				err = target.CloseTarget(v.TargetID).Do(cdp.WithExecutor(c2, c.Browser))
				if err != nil {
					fmt.Println(err)
				}
				time.Sleep(time.Millisecond * 200)
			}
		}
	}

	return &Ctx{
		Context: c2,
		Cancel:  f2,
	}
}

// 打开新的tab
func NewTabContext(ctx *Ctx, url string) *Ctx {
	var ts = httptest.NewServer(http.NewServeMux())
	defer ts.Close()
	var ch = chromedp.WaitNewTarget(ctx, func(info *target.Info) bool {
		return info.URL != ""
	})

	_ = chromedp.Run(ctx, chromedp.Evaluate("window.open('', '_blank')", nil))
	var c, f = chromedp.NewContext(ctx, chromedp.WithTargetID(<-ch))

	var newCtx = &Ctx{Context: c, Cancel: f}
	if strings.HasPrefix(url, "http") {
		_ = chromedp.Run(newCtx, chromedp.Navigate(url))
	}
	return newCtx
}

// 根据页面标题，查找tab页面；不存在则新开一个tab
func FindTabContext(ctx *Ctx, link string) *Ctx {
	var tabCtx = &Ctx{}
	var targets, err = chromedp.Targets(ctx)
	var have = false
	if err == nil && targets != nil && len(targets) > 0 {
		for _, target := range targets {
			if strings.Contains(target.URL, link) {
				have = true
				var c, f = chromedp.NewContext(ctx, chromedp.WithTargetID(target.TargetID))
				tabCtx.Context = c
				tabCtx.Cancel = f
				chromedp.Run(tabCtx, chromedp.Reload())
			}
		}
	}

	if have == false {
		tabCtx = NewTabContext(ctx, link)
	}
	return tabCtx
}

// 注册新tab标签的监听服务
func NewTabListener(ctx context.Context) <-chan target.ID {
	mux := http.NewServeMux()
	ts := httptest.NewServer(mux)
	defer ts.Close()

	return chromedp.WaitNewTarget(ctx, func(info *target.Info) bool {
		return info.URL != ""
	})
}

// 关闭远程浏览器
func CloseRemoteContext(ip string, port int) {
	var ctx, fun = chromedp.NewRemoteAllocator(context.Background(), fmt.Sprintf("ws://%s:%d/", ip, port))
	if ctx == nil || fun == nil {
		return
	}

	//创建上线文
	ctx, fun = chromedp.NewContext(ctx, chromedp.WithLogf(log.Printf))
	var c = chromedp.FromContext(ctx)

	//获取tab页
	time.Sleep(time.Millisecond * 500)
	var targets, err = chromedp.Targets(ctx)
	if err != nil {
		return
	}

	for _, v := range targets {
		err = target.CloseTarget(v.TargetID).Do(cdp.WithExecutor(ctx, c.Browser))
		if err != nil {
			fmt.Println(err)
		}
		time.Sleep(time.Millisecond * 200)
		break
	}
}
