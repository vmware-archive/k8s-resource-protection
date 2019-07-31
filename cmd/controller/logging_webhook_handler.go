package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type ExtendedHandler interface {
	webhook.AdmissionHandler
	InjectClient(client.Client) error
	InjectDecoder(*admission.Decoder) error
}

type loggingWebhookHandler struct {
	Handler ExtendedHandler
	Log     logr.Logger
	Debug   bool
}

func (v loggingWebhookHandler) Handle(ctx context.Context, req admission.Request) admission.Response {
	resp := v.Handler.Handle(ctx, req)
	if v.Debug {
		reqBs, _ := json.Marshal(req)
		respBs, _ := json.Marshal(resp)
		v.Log.Info(fmt.Sprintf("req: %s resp: %s\n", reqBs, respBs))
	}
	return resp
}

func (v loggingWebhookHandler) InjectClient(c client.Client) error {
	return v.Handler.InjectClient(c)
}

func (v loggingWebhookHandler) InjectDecoder(d *admission.Decoder) error {
	return v.Handler.InjectDecoder(d)
}
