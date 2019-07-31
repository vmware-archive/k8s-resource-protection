package main

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	prefixAnnNs             = "rp.k14s.io/"
	allowedOperationsAnnKey = prefixAnnNs + "allowed-operations" // valid values: "" (none are allowed), "CREATE,UPDATE,DELETE,CONNECT"
)

var (
	knownAnnKeys = map[string]struct{}{
		allowedOperationsAnnKey: struct{}{},
	}
)

func HasUnknownAnns(res *unstructured.Unstructured) *admission.Response {
	for key, _ := range res.GetAnnotations() {
		if strings.HasPrefix(key, prefixAnnNs) {
			if _, found := knownAnnKeys[key]; !found {
				resp := admission.Denied(fmt.Sprintf("unknown annotation '%s' with prefix '%s'", key, prefixAnnNs))
				return &resp
			}
		}
	}
	return nil
}
