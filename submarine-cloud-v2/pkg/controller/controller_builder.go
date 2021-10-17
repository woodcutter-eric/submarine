/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package controller

import (
	submarinescheme "github.com/apache/submarine/submarine-cloud-v2/pkg/client/clientset/versioned/scheme"
	corev1 "k8s.io/api/core/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
)

type Builder interface {
	Build() ControllerInterface
}

type ControllerFunc func()

// Main controller builder
type ControllerBuilder struct {
	controller *Controller
	config     *BuilderConfig
	actions    map[string]ControllerFunc
}

func NewControllerBuilder(config *BuilderConfig) Builder {
	return &ControllerBuilder{
		controller: &Controller{},
		config:     config,
		actions:    map[string]ControllerFunc{},
	}
}

func (cb *ControllerBuilder) Incluster(
	incluster bool,
) *ControllerBuilder {
	cb.controller.incluster = incluster
	return cb
}

func (cb *ControllerBuilder) Build() ControllerInterface {
	cb.Initailize()
	cb.AddClientsets()
	cb.AddListers()
	cb.RegisterEventHandlers()
	cb.AddEventHandlers()

	return cb.controller
}

func (cb *ControllerBuilder) Initailize() *ControllerBuilder {
	// Add Submarine types to the default Kubernetes Scheme so Events can be
	// logged for Submarine types.
	utilruntime.Must(submarinescheme.AddToScheme(scheme.Scheme))
	klog.V(4).Info("Creating event broadcaster")

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartStructuredLogging(0)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: cb.config.kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: controllerAgentName})

	workqueue := workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Submarines")

	cb.controller.incluster = cb.config.incluster
	cb.controller.recorder = recorder
	cb.controller.workqueue = workqueue

	return cb
}

func (cb *ControllerBuilder) AddClientsets() *ControllerBuilder {
	cb.controller.kubeclientset = cb.config.kubeclientset
	cb.controller.submarineclientset = cb.config.submarineclientset
	cb.controller.traefikclientset = cb.config.traefikclientset

	return cb
}

func (cb *ControllerBuilder) AddListers() *ControllerBuilder {
	cb.controller.submarinesLister = cb.config.submarineInformer.Lister()
	cb.controller.submarinesSynced = cb.config.submarineInformer.Informer().HasSynced

	cb.controller.deploymentLister = cb.config.deploymentInformer.Lister()
	cb.controller.namespaceLister = cb.config.namespaceInformer.Lister()
	cb.controller.serviceLister = cb.config.serviceInformer.Lister()
	cb.controller.serviceaccountLister = cb.config.serviceaccountInformer.Lister()
	cb.controller.persistentvolumeclaimLister = cb.config.persistentvolumeclaimInformer.Lister()
	cb.controller.ingressLister = cb.config.ingressInformer.Lister()
	cb.controller.ingressrouteLister = cb.config.ingressrouteInformer.Lister()
	cb.controller.roleLister = cb.config.roleInformer.Lister()
	cb.controller.rolebindingLister = cb.config.rolebindingInformer.Lister()

	return cb
}

func (cb *ControllerBuilder) RegisterEventHandlers() *ControllerBuilder {
	// Setting up event handler for Submarine
	cb.RegisterSubmarineEventHandlers()

	// Setting up event handler for other resources
	cb.RegisterNamespaceEventHandlers()
	cb.RegisterDeploymentEventHandlers()
	cb.RegisterServiceEventHandlers()
	cb.RegisterServiceAccountEventHandlers()
	cb.RegisterPersistentVolumeClaimEventHandlers()
	cb.RegisterIngressEventHandlers()
	cb.RegisterIngressRouteEventHandlers()
	cb.RegisterRoleEventHandlers()
	cb.RegisterRoleBindingEventHandlers()

	return cb
}

func (cb *ControllerBuilder) AddEventHandlers() *ControllerBuilder {
	klog.Info("Setting up event handlers")

	for _, action := range cb.actions {
		go action()
	}

	return cb
}
