package crawler

import (
	"context"
	"github.com/chromedp/chromedp"
	"github.com/chromedp/chromedp/kb"
	"github.com/saxon134/go-utils/saData"
	"github.com/saxon134/go-utils/saData/saError"
	"strings"
	"time"
)

type Step struct {
	Act string `json:"act"`
	Sel string `json:"sel"`
	Ms  int64  `json:"ms"` //毫秒
	Val string `json:"val"`
}

func RunSteps(ctx context.Context, steps []*Step) error {
	var err error
	for _, step := range steps {
		err = step.Run(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *Step) Run(ctx context.Context) error {
	var contents = strings.Split(m.Val, ",")
	switch m.Act {
	case "sleep":
		if m.Ms > 0 {
			time.Sleep(time.Millisecond * time.Duration(m.Ms))
		}
	case "wait":
		if m.Sel != "" {
			VisibleWithTimeout(ctx, m.Sel, time.Millisecond*time.Duration(m.Ms))
		}
	case "navigate":
		if strings.HasPrefix(m.Val, "http") {
			return chromedp.Run(ctx, chromedp.Navigate(m.Val))
		}
	case "reload":
		return chromedp.Run(ctx, chromedp.Reload())
	case "click":
		if m.Sel != "" {
			Click(ctx, m.Sel, contents...)
		}
	case "input":
		if m.Sel != "" {
			var selector = findSelector(ctx, m.Sel, contents)
			if selector != "" {
				Run(ctx, chromedp.WaitReady(selector))
				time.Sleep(time.Millisecond * 100)

				var err = chromedp.Run(ctx,
					chromedp.Focus(selector),
					chromedp.SendKeys(selector, m.Val),
					chromedp.Sleep(time.Second*time.Duration((saData.StrLen(m.Val)-1)/2+1)),
					chromedp.Blur(selector),
				)
				if err == nil {
					return nil
				}
			}
		}
	case "fill":
		if m.Sel != "" {
			var selector = findSelector(ctx, m.Sel, contents)
			if selector != "" {
				time.Sleep(time.Millisecond * 300)
				Run(ctx, chromedp.WaitReady(selector))

				var c, _ = context.WithTimeout(ctx, time.Second)
				var err = chromedp.Run(c,
					chromedp.Focus(selector),
					chromedp.SetValue(selector, m.Val),
					chromedp.Sleep(time.Millisecond*50),
					chromedp.SendKeys(selector, " "),
					chromedp.Sleep(time.Millisecond*50),
					chromedp.SendKeys(selector, kb.Backspace),
					chromedp.Sleep(time.Millisecond*50),
					chromedp.Blur(selector),
				)
				if err == nil {
					time.Sleep(time.Millisecond * 50)
					return nil
				} else {
					time.Sleep(time.Millisecond * 50)
					if InnerText(ctx, m.Sel, contents...) == m.Val {
						c, _ = context.WithTimeout(ctx, time.Second)
						_ = chromedp.Run(c,
							chromedp.Focus(selector),
							chromedp.Sleep(time.Millisecond*50),
							chromedp.Blur(selector),
							chromedp.Sleep(time.Millisecond*50),
						)
						return nil
					}
				}

				c, _ = context.WithTimeout(ctx, time.Second*time.Duration((saData.StrLen(m.Val)-1)/2+1))
				err = chromedp.Run(c,
					chromedp.Clear(selector),
					chromedp.SendKeys(selector, m.Val),
				)
				if err == nil {
					time.Sleep(time.Millisecond * 50)
					return nil
				}

				c, _ = context.WithTimeout(ctx, time.Second)
				err = chromedp.Run(c,
					chromedp.Focus(selector),
					chromedp.SetValue(selector, m.Val),
					chromedp.Blur(selector),
				)
				if err == nil {
					time.Sleep(time.Millisecond * 50)
					return nil
				}
				return err
			}
		}
	case "upload":
		var selector = findSelector(ctx, m.Sel, contents)
		if selector == "" {
			return saError.New("[Crawler Step: upload err, selector not exist.]" + m.Sel)
		}

		var files = strings.Split(m.Val, ",")
		if len(files) == 0 {
			return saError.New("[Crawler Step: upload err, file is empty.]" + m.Sel)
		}
		return chromedp.SetUploadFiles(selector, files, chromedp.ByQuery).Do(ctx)
	default:
		return saError.New("[Crawler Step: action not exist, " + m.Act)
	}
	return nil
}
