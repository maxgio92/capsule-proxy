// Copyright 2020-2021 Clastix Labs
// SPDX-License-Identifier: Apache-2.0

package storageclass

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-logr/logr"
	"github.com/gorilla/mux"
	v1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/clastix/capsule-proxy/internal/modules"
	"github.com/clastix/capsule-proxy/internal/modules/errors"
	"github.com/clastix/capsule-proxy/internal/request"
	"github.com/clastix/capsule-proxy/internal/tenant"
)

type get struct {
	client client.Client
	log    logr.Logger
}

func Get(client client.Client) modules.Module {
	return &get{client: client, log: ctrl.Log.WithName("storageclass_get")}
}

func (g get) Path() string {
	return "/apis/storage.k8s.io/v1/storageclasses/{name}"
}

func (g get) Methods() []string {
	return []string{}
}

func (g get) Handle(proxyTenants []*tenant.ProxyTenant, proxyRequest request.Request) (selector labels.Selector, err error) {
	httpRequest := proxyRequest.GetHTTPRequest()

	_, exactMatch, regexMatch := getStorageClasses(httpRequest, proxyTenants)

	name := mux.Vars(httpRequest)["name"]

	sc := &v1.StorageClassList{}
	if err = g.client.List(context.Background(), sc, client.MatchingLabels{"name": name}); err != nil {
		return nil, errors.NewBadRequest(
			err,
			&metav1.StatusDetails{
				Group: "storage.k8s.io",
				Kind:  "storageclasses",
			},
		)
	}

	var r *labels.Requirement
	r, err = getStorageClassSelector(sc, exactMatch, regexMatch)

	switch {
	case err == nil:
		return labels.NewSelector().Add(*r), nil
	case httpRequest.Method == http.MethodGet:
		return nil, errors.NewNotFoundError(
			fmt.Sprintf("storageclasses.storage.k8s.io \"%s\" not found", name),
			&metav1.StatusDetails{
				Name:  name,
				Group: "storage.k8s.io",
				Kind:  "storageclasses",
			},
		)
	default:
		return nil, nil
	}
}
