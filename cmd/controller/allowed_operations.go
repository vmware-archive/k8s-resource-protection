package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var (
	allAllowedOperations = map[admissionv1beta1.Operation]struct{}{
		admissionv1beta1.Create:  struct{}{},
		admissionv1beta1.Update:  struct{}{},
		admissionv1beta1.Delete:  struct{}{},
		admissionv1beta1.Connect: struct{}{},
	}
)

type allowedOperationsValidator struct {
	client  client.Client
	decoder *admission.Decoder
}

var _ ExtendedHandler = &allowedOperationsValidator{}

func (v *allowedOperationsValidator) Handle(ctx context.Context, req admission.Request) admission.Response {
	res := &unstructured.Unstructured{}

	// TODO more correct way to convert?
	res.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   req.Kind.Group,
		Version: req.Kind.Version,
		Kind:    req.Kind.Kind,
	})

	if len(req.Object.Raw) > 0 {
		err := v.decoder.Decode(req, res)
		if err != nil {
			return admission.Errored(http.StatusBadRequest, fmt.Errorf("decoding: %s", err))
		}
	} else {
		err := v.client.Get(ctx, types.NamespacedName{Namespace: req.Namespace, Name: req.Name}, res)
		if err != nil {
			return admission.Errored(http.StatusBadRequest, fmt.Errorf("getting resource: %s", err))
		}
	}

	obj := v.newObjFromResource(res)

	resp := HasUnknownAnns(res)
	if resp != nil {
		return *resp
	}

	switch req.Operation {
	case admissionv1beta1.Create:
		allowedOps, resp := v.canPerform(obj, admissionv1beta1.Create)
		if resp != nil {
			return *resp
		}

		return v.newDeny(admissionv1beta1.Create, allowedOps)

	case admissionv1beta1.Update:
		resp := v.validateOps(obj)
		if resp != nil {
			return *resp
		}

		oldAllowedOps, resp := v.canPerform(v.newObjFromObj(req.OldObject), admissionv1beta1.Update)
		if resp != nil {
			return *resp
		}

		newAllowedOps, resp := v.canPerform(obj, admissionv1beta1.Update)
		if resp != nil {
			return *resp
		}

		return v.newDeny(admissionv1beta1.Update, append(oldAllowedOps, newAllowedOps...))

	case admissionv1beta1.Delete:
		allowedOps, resp := v.canPerform(obj, admissionv1beta1.Delete)
		if resp != nil {
			return *resp
		}

		return v.newDeny(admissionv1beta1.Delete, allowedOps)

	// TODO what is connect?
	case admissionv1beta1.Connect:
		allowedOps, resp := v.canPerform(obj, admissionv1beta1.Connect)
		if resp != nil {
			return *resp
		}

		return v.newDeny(admissionv1beta1.Connect, allowedOps)

	default:
		return admission.Denied(fmt.Sprintf("unknown operation '%s'", req.Operation))
	}
}

func (v *allowedOperationsValidator) newDeny(wantedOp admissionv1beta1.Operation, allowedOps []admissionv1beta1.Operation) admission.Response {
	var ops []string
	uniqOps := map[admissionv1beta1.Operation]struct{}{}
	for _, op := range allowedOps {
		if _, found := uniqOps[op]; !found {
			uniqOps[op] = struct{}{}
			ops = append(ops, string(op))
		}
	}

	return admission.Denied(fmt.Sprintf("resource %s is denied via '%s' annotation (allows only %s)",
		wantedOp, allowedOperationsAnnKey, strings.Join(ops, ", ")))
}

type parsedObjFunc func() (*unstructured.Unstructured, *admission.Response)

func (v *allowedOperationsValidator) newObjFromResource(res *unstructured.Unstructured) parsedObjFunc {
	return func() (*unstructured.Unstructured, *admission.Response) { return res, nil }
}

func (v *allowedOperationsValidator) newObjFromObj(obj runtime.RawExtension) parsedObjFunc {
	return func() (*unstructured.Unstructured, *admission.Response) {
		res := &unstructured.Unstructured{}

		err := v.decoder.DecodeRaw(obj, res)
		if err != nil {
			resp := admission.Errored(http.StatusBadRequest, fmt.Errorf("decoding raw: %s", err))
			return nil, &resp
		}

		return res, nil
	}
}

func (v *allowedOperationsValidator) validateOps(obj parsedObjFunc) *admission.Response {
	_, resp := v.allowedOps(obj)
	return resp
}

func (v *allowedOperationsValidator) allowedOps(obj parsedObjFunc) ([]admissionv1beta1.Operation, *admission.Response) {
	res, resp := obj()
	if resp != nil {
		return nil, resp
	}

	annVal, found := res.GetAnnotations()[allowedOperationsAnnKey]
	if !found {
		return []admissionv1beta1.Operation{
			admissionv1beta1.Create,
			admissionv1beta1.Update,
			admissionv1beta1.Delete,
			admissionv1beta1.Connect,
		}, nil
	}

	if annVal == "" {
		// No operations are allowed
		return nil, nil
	}

	var allowedOps []admissionv1beta1.Operation

	for _, op := range strings.Split(annVal, ",") {
		if _, found := allAllowedOperations[admissionv1beta1.Operation(op)]; !found {
			resp := admission.Denied(fmt.Sprintf("annotation '%s' contains unknown allowed operation '%s'", allowedOperationsAnnKey, op))
			return nil, &resp
		}
		allowedOps = append(allowedOps, admissionv1beta1.Operation(op))
	}

	return allowedOps, nil
}

func (v *allowedOperationsValidator) canPerform(obj parsedObjFunc, wantedOp admissionv1beta1.Operation) ([]admissionv1beta1.Operation, *admission.Response) {
	allowedOps, resp := v.allowedOps(obj)
	if resp != nil {
		return allowedOps, resp
	}

	for _, op := range allowedOps {
		if op == wantedOp {
			resp := admission.Allowed(fmt.Sprintf("operation '%s' is allowed", op))
			return allowedOps, &resp
		}
	}

	return allowedOps, nil
}

func (v *allowedOperationsValidator) InjectClient(c client.Client) error {
	v.client = c
	return nil
}

func (v *allowedOperationsValidator) InjectDecoder(d *admission.Decoder) error {
	v.decoder = d
	return nil
}
