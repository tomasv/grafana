package notifications

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"

	"golang.org/x/net/context/ctxhttp"

	"github.com/grafana/grafana/pkg/log"
	"github.com/grafana/grafana/pkg/util"
)

type Webhook struct {
	Url        string
	User       string
	Password   string
	Body       string
	HttpMethod string
}

var (
	webhookQueue chan *Webhook
	webhookLog   log.Logger
)

func initWebhookQueue() {
	webhookLog = log.New("notifications.webhook")
	webhookQueue = make(chan *Webhook, 10)
	go processWebhookQueue()
}

func processWebhookQueue() {
	for {
		select {
		case webhook := <-webhookQueue:
			err := sendWebRequestSync(context.TODO(), webhook)

			if err != nil {
				webhookLog.Error("Failed to send webrequest ", "error", err)
			}
		}
	}
}

func sendWebRequestSync(ctx context.Context, webhook *Webhook) error {
	webhookLog.Debug("Sending webhook", "url", webhook.Url, "http method", webhook.HttpMethod)

	if webhook.HttpMethod == "" {
		webhook.HttpMethod = http.MethodPost
	}

	request, err := http.NewRequest(webhook.HttpMethod, webhook.Url, bytes.NewReader([]byte(webhook.Body)))
	if err != nil {
		return err
	}

	request.Header.Add("Content-Type", "application/json")
	request.Header.Add("User-Agent", "Grafana")
	if webhook.User != "" && webhook.Password != "" {
		request.Header.Add("Authorization", util.GetBasicAuthHeader(webhook.User, webhook.Password))
	}

	resp, err := ctxhttp.Do(ctx, http.DefaultClient, request)
	if err != nil {
		return err
	}

	if resp.StatusCode/100 == 2 {
		return nil
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	webhookLog.Debug("Webhook failed", "statuscode", resp.Status, "body", string(body))
	return fmt.Errorf("Webhook response status %v", resp.Status)
}

var addToWebhookQueue = func(msg *Webhook) {
	webhookQueue <- msg
}
