// SPDX-FileCopyrightText: 2024 The Crossplane Authors <https://crossplane.io>
//
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/crossplane/upjet/v2/pkg/controller"

	authorizationmodel "github.com/l-nmch/provider-openfga/internal/controller/namespaced/model/authorizationmodel"
	providerconfig "github.com/l-nmch/provider-openfga/internal/controller/namespaced/providerconfig"
	store "github.com/l-nmch/provider-openfga/internal/controller/namespaced/store/store"
	relationshiptuple "github.com/l-nmch/provider-openfga/internal/controller/namespaced/tuple/relationshiptuple"
)

// Setup creates all controllers with the supplied logger and adds them to
// the supplied manager.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	for _, setup := range []func(ctrl.Manager, controller.Options) error{
		authorizationmodel.Setup,
		providerconfig.Setup,
		store.Setup,
		relationshiptuple.Setup,
	} {
		if err := setup(mgr, o); err != nil {
			return err
		}
	}
	return nil
}

// SetupGated creates all controllers with the supplied logger and adds them to
// the supplied manager gated.
func SetupGated(mgr ctrl.Manager, o controller.Options) error {
	for _, setup := range []func(ctrl.Manager, controller.Options) error{
		authorizationmodel.SetupGated,
		providerconfig.SetupGated,
		store.SetupGated,
		relationshiptuple.SetupGated,
	} {
		if err := setup(mgr, o); err != nil {
			return err
		}
	}
	return nil
}

// SetupWebhookWithManager registers conversion webhooks for all resource kinds in the group.
func SetupWebhookWithManager(mgr ctrl.Manager) error {
	for _, setup := range []func(ctrl.Manager) error{
		authorizationmodel.SetupWebhookWithManager,
		providerconfig.SetupWebhookWithManager,
		store.SetupWebhookWithManager,
		relationshiptuple.SetupWebhookWithManager,
	} {
		if err := setup(mgr); err != nil {
			return err
		}
	}
	return nil
}
