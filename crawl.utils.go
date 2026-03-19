package crawl

import (
	"context"
	"fmt"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"github.com/saxon134/go-utils/saData"
	"github.com/saxon134/go-utils/saData/saHit"
	"github.com/saxon134/go-utils/saLog"
	"strings"
	"time"
)

func Run(ctx context.Context, actions ...chromedp.Action) {
	if ctx == nil {
		return
	}
	_ = chromedp.Run(ctx, actions...)
}

func Click(ctx context.Context, selector string, contents ...string) {
	time.Sleep(time.Millisecond * 100)
	var sel = findSelector(ctx, selector, contents)
	if sel != "" {
		var nodes = NodesWithTimeout(ctx, sel, 0)
		if len(nodes) > 0 {
			NodeClick(ctx, nodes[len(nodes)-1])
		}
		return
	}

	//最后还是得点击一下，哪怕卡在这里
	saLog.Err("[Crawler Click Block] " + selector)
	Run(ctx, chromedp.Click(selector))
}

func Value(ctx context.Context, selector string, contents ...string) string {
	selector = findSelector(ctx, selector, contents)
	if selector != "" {
		var nodes = NodesWithTimeout(ctx, selector, 0)
		if len(nodes) == 0 {
			return ""
		}

		var str = saHit.OrStr(nodes[0].NodeValue, nodes[0].Value)
		if str != "" {
			return str
		}

		if len(nodes[0].Children) > 0 {
			str = saHit.OrStr(nodes[0].Children[0].NodeValue, nodes[0].Children[0].Value)
			return str
		}
	}
	return ""
}

func InnerText(ctx context.Context, selector string, contents ...string) string {
	selector = findSelector(ctx, selector, contents)
	if selector != "" {
		var str = ""
		selector = fmt.Sprintf(`document.querySelector("%s").innerText`, selector)
		Run(ctx, chromedp.Evaluate(selector, &str))
		return str
	}
	return ""
}

// duration: -1表示永远等待
func VisibleWithTimeout(ctx context.Context, selector string, duration time.Duration) (visible bool) {
	if ctx == nil || selector == "" {
		return false
	}

	var nodes []*cdp.Node
	var startTime = time.Now()
	for {
		Run(
			ctx,
			chromedp.Query(selector, chromedp.AtLeast(0), chromedp.After(func(ctx context.Context, id runtime.ExecutionContextID, node ...*cdp.Node) error {
				nodes = node
				return nil
			})),
		)
		if len(nodes) > 0 {
			return true
		}

		if duration <= 0 || time.Now().After(startTime.Add(duration)) {
			return false
		}

		if duration == 0 {
			return false
		}

		if duration > 0 && time.Now().After(startTime.Add(duration)) {
			return false
		}

		if duration < 0 {
			time.Sleep(time.Second * 3)
		} else {
			var t = duration / 10
			if t < time.Millisecond*100 {
				t = time.Millisecond * 100
			}
			time.Sleep(t)
		}
	}
}

func NodeClick(ctx context.Context, node *cdp.Node) {
	if node != nil {
		Run(ctx, chromedp.MouseClickNode(node))
	}
}

func NodeValue(node *cdp.Node) string {
	if node == nil {
		return ""
	}

	var str = saHit.OrStr(node.NodeValue, node.Value)
	if str != "" {
		return str
	}

	if len(node.Children) > 0 {
		str = saHit.OrStr(node.Children[0].NodeValue, node.Children[0].Value)
		return str
	}

	return ""
}

func NodeAttributeExisted(node *cdp.Node, attribute string) bool {
	if node == nil || attribute == "" {
		return false
	}

	if node.Attributes == nil || len(node.Attributes) == 0 {
		return node.AttributeValue(attribute) != ""
	}

	for _, v := range node.Attributes {
		if strings.Contains(v, attribute) {
			return true
		}
	}
	return false
}

func NodeAttributeValue(node *cdp.Node, attribute string) string {
	if node == nil || attribute == "" {
		return ""
	}

	if node.Attributes == nil || len(node.Attributes) == 0 {
		return node.AttributeValue(attribute)
	}
	return ""
}

func NodesWithTimeout(ctx context.Context, sel string, duration time.Duration) (nodes []*cdp.Node) {
	var now = time.Now()
	for {
		Run(
			ctx,
			chromedp.Query(sel, chromedp.AtLeast(0), chromedp.After(func(ctx context.Context, id runtime.ExecutionContextID, node ...*cdp.Node) error {
				nodes = node
				return nil
			})),
		)
		if len(nodes) > 0 {
			return nodes
		}

		if duration <= 0 || time.Now().After(now.Add(duration)) {
			return
		}

		var t = duration / 10
		if t < time.Millisecond*100 {
			t = time.Millisecond * 100
		}
		time.Sleep(t)
	}
}

func GetCookie(ctx *Ctx) {
	Run(
		ctx,
		chromedp.ActionFunc(func(c context.Context) error {
			var cookies, err = network.GetCookies().Do(c)
			if err != nil {
				saLog.Err(err)
				return err
			}

			var cookie = ""
			for _, v := range cookies {
				cookie = cookie + v.Name + "=" + v.Value + ";"
			}
			if cookie != "" {
				ctx.Cookie = cookie
			}

			return nil
		}),
	)
}

func findSelector(ctx context.Context, selector string, contents []string) string {
	if ctx == nil || selector == "" {
		panic("selector error")
	}

	if VisibleWithTimeout(ctx, selector, 0) {
		return selector
	}

	if len(contents) == 0 {
		return ""
	}

	var divAry = strings.Split(selector, ">")
	if len(divAry) >= 5 {

		//尝试减少1-2个div，是否可以匹配到
		var tmpAry = make([]string, 0, len(divAry)+3)
		tmpAry = append(tmpAry, divAry...)
		for i := 0; i < 2; i++ {
			tmpAry = append(tmpAry[0:len(tmpAry)-1], tmpAry[len(tmpAry)-1])
			var sel = strings.Join(tmpAry, " > ")
			var text = InnerText(ctx, sel)
			if saData.Contains(text, contents) {
				return text
			}
		}

		//尝试增加1-3个div,是否可以匹配到
		tmpAry = append(tmpAry, divAry...)
		for i := 0; i < 3; i++ {
			divAry = append(tmpAry[0:len(tmpAry)-1], " div ", tmpAry[len(tmpAry)-1])
			var sel = strings.Join(tmpAry, ">")
			var text = InnerText(ctx, sel)
			if saData.Contains(text, contents) {
				return text
			}
		}
	}

	return ""
}
