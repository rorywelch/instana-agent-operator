/*
(c) Copyright IBM Corp. 2024, 2025

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	namespaces_configmap "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/agent/configmap/namespaces-configmap"
	agentdaemonset "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/agent/daemonset"
	headlessservice "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/agent/headless-service"
	agentrbac "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/agent/rbac"
	agentsecrets "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/agent/secrets"
	keyssecret "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/agent/secrets/keys-secret"
	tlssecret "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/agent/secrets/tls-secret"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/agent/service"
	agentserviceaccount "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/agent/serviceaccount"
	backends "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/backends"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/builder"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/namespaces"
	k8ssensorconfigmap "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/k8s-sensor/configmap"
	k8ssensordeployment "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/k8s-sensor/deployment"
	k8ssensorpoddisruptionbudget "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/k8s-sensor/poddisruptionbudget"
	k8ssensorrbac "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/k8s-sensor/rbac"
	k8ssensorserviceaccount "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/k8s-sensor/serviceaccount"
	"github.com/instana/instana-agent-operator/pkg/k8s/operator/operator_utils"
	"github.com/instana/instana-agent-operator/pkg/k8s/operator/status"
	corev1 "k8s.io/api/core/v1"
)

func getDaemonSetBuilders(
	agent *instanav1.InstanaAgent,
	isOpenShift bool,
	statusManager status.AgentStatusManager,
) []builder.ObjectBuilder {
	if len(agent.Spec.Zones) == 0 {
		return []builder.ObjectBuilder{agentdaemonset.NewDaemonSetBuilder(agent, isOpenShift, statusManager)}
	}

	builders := make([]builder.ObjectBuilder, 0, len(agent.Spec.Zones))

	for _, zone := range agent.Spec.Zones {
		builders = append(
			builders,
			agentdaemonset.NewDaemonSetBuilderWithZoneInfo(agent, isOpenShift, statusManager, &zone),
		)
	}

	return builders
}

func getK8sSensorDeployments(
	agent *instanav1.InstanaAgent,
	isOpenShift bool,
	statusManager status.AgentStatusManager,
	k8SensorBackends []backends.K8SensorBackend,
	keysSecret *corev1.Secret,
) []builder.ObjectBuilder {
	builders := make([]builder.ObjectBuilder, 0, len(k8SensorBackends))

	for _, backend := range k8SensorBackends {
		builders = append(
			builders,
			k8ssensordeployment.NewDeploymentBuilder(agent, isOpenShift, statusManager, backend, keysSecret),
		)
	}

	return builders
}

func (r *InstanaAgentReconciler) applyResources(
	ctx context.Context,
	agent *instanav1.InstanaAgent,
	isOpenShift bool,
	operatorUtils operator_utils.OperatorUtils,
	statusManager status.AgentStatusManager,
	keysSecret *corev1.Secret,
	k8SensorBackends []backends.K8SensorBackend,
	namespacesDetails namespaces.NamespacesDetails,
) reconcileReturn {
	log := r.loggerFor(ctx, agent)
	log.V(1).Info("applying Kubernetes resources for agent")

	builders := append(
		getDaemonSetBuilders(agent, isOpenShift, statusManager),
		headlessservice.NewHeadlessServiceBuilder(agent),
		agentsecrets.NewConfigBuilder(agent, statusManager, keysSecret, k8SensorBackends),
		agentsecrets.NewContainerBuilder(agent, keysSecret),
		tlssecret.NewSecretBuilder(agent),
		service.NewServiceBuilder(agent),
		agentrbac.NewClusterRoleBuilder(agent),
		agentrbac.NewClusterRoleBindingBuilder(agent),
		agentserviceaccount.NewServiceAccountBuilder(agent),
		k8ssensorpoddisruptionbudget.NewPodDisruptionBudgetBuilder(agent),
		k8ssensorrbac.NewClusterRoleBuilder(agent),
		k8ssensorrbac.NewClusterRoleBindingBuilder(agent),
		k8ssensorserviceaccount.NewServiceAccountBuilder(agent),
		k8ssensorconfigmap.NewConfigMapBuilder(agent, k8SensorBackends),
		keyssecret.NewSecretBuilder(agent, k8SensorBackends),
		namespaces_configmap.NewConfigMapBuilder(agent, statusManager, namespacesDetails),
	)

	builders = append(builders, getK8sSensorDeployments(agent, isOpenShift, statusManager, k8SensorBackends, keysSecret)...)

	if err := operatorUtils.ApplyAll(builders...); err != nil {
		log.Error(err, "failed to apply kubernetes resources for agent")
		return reconcileFailure(err)
	}

	log.V(1).Info("successfully applied kubernetes resources for agent")
	return reconcileContinue()
}
