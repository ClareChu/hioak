// Copyright 2018 John Deng (hi.devops.io@gmail.com).
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package openshift

import (
	"fmt"
	"github.com/jinzhu/copier"
	"github.com/openshift/api/apps/v1"
	appsv1 "github.com/openshift/client-go/apps/clientset/versioned/typed/apps/v1"
	"hidevops.io/hiboot/pkg/log"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
)

type DeploymentConfig struct {
	clientSet appsv1.AppsV1Interface
}

func newDeploymentConfig(clientSet appsv1.AppsV1Interface) *DeploymentConfig {
	log.Debug("NewDeploymentConfig()")
	return &DeploymentConfig{
		clientSet: clientSet,
	}
}

type DeploymentRequest struct {
	Name           string
	Namespace      string
	FullName       string
	Version        string
	Env            interface{}
	Labels         map[string]string
	Ports          interface{}
	Replicas       int32
	Force          bool
	HealthEndPoint string
	NodeSelector   string
	Tag            string
}

func (dc *DeploymentConfig) Create(request *DeploymentRequest) error {
	log.Debug("DeploymentConfig.Create()", request.Force)
	// env
	e := make([]corev1.EnvVar, 0)
	copier.Copy(&e, request.Env)
	selector := map[string]string{}
	if request.NodeSelector != "" {
		selector[strings.Split(request.NodeSelector, "=")[0]] = strings.Split(request.NodeSelector, "=")[1]
	}
	p := make([]corev1.ContainerPort, 0)
	copier.Copy(&p, request.Ports)
	cfg := &v1.DeploymentConfig{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps.openshift.io/v1",
			Kind:       "DeploymentConfig",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   request.FullName,
			Labels: request.Labels,
		},
		Spec: v1.DeploymentConfigSpec{
			Replicas: request.Replicas,

			Selector: map[string]string{
				"app":     request.Name,
				"version": request.Version,
			},

			Strategy: v1.DeploymentStrategy{
				Type: v1.DeploymentStrategyTypeRolling,
			},

			Template: &corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   request.Name,
					Labels: request.Labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Env:             e,
							Image:           " ",
							ImagePullPolicy: corev1.PullAlways,
							Name:            request.Name,
							Ports:           p,
							ReadinessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									Exec: &corev1.ExecAction{
										Command: []string{
											"curl",
											"--silent",
											"--show-error",
											"--fail",
											request.HealthEndPoint,
										},
									},
								},
								InitialDelaySeconds: 60,
								TimeoutSeconds:      10,
								PeriodSeconds:       60,
							},
							LivenessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									Exec: &corev1.ExecAction{
										Command: []string{
											"curl",
											"--silent",
											"--show-error",
											"--fail",
											request.HealthEndPoint,
										},
									},
								},
								InitialDelaySeconds: 60,
								TimeoutSeconds:      10,
								PeriodSeconds:       60,
							},
						},
					},
					DNSPolicy:     corev1.DNSClusterFirst,
					RestartPolicy: corev1.RestartPolicyAlways,
					SchedulerName: "default-scheduler",
					NodeSelector:  selector,
				},
			},
			Test: false,
			Triggers: v1.DeploymentTriggerPolicies{
				{
					Type: v1.DeploymentTriggerOnImageChange,
					ImageChangeParams: &v1.DeploymentTriggerImageChangeParams{
						Automatic: true,
						ContainerNames: []string{
							request.Name,
						},
						From: corev1.ObjectReference{
							Kind:      "ImageStreamTag",
							Name:      request.Name + ":" + request.Tag,
							Namespace: request.Namespace,
						},
					},
				},
			},
		},
	}

	result, err := dc.clientSet.DeploymentConfigs(request.Namespace).Get(request.FullName, metav1.GetOptions{})
	switch {
	case err == nil:
		// select update or patch according to the user's request
		if request.Force {
			cfg.ObjectMeta.ResourceVersion = result.ResourceVersion
			result, err = dc.clientSet.DeploymentConfigs(request.Namespace).Update(cfg)
			if err == nil {
				log.Infof("Updated DeploymentConfig %v.", result.Name)
				_, err := dc.Instantiate(request.Name, request.Namespace, request.FullName)
				if err != nil {
					log.Error(err.Error())
				}
				return err
			} else {
				return err
			}
		}
	case errors.IsNotFound(err):
		_, err := dc.clientSet.DeploymentConfigs(request.Namespace).Create(cfg)
		if err != nil {
			return err
		}
		log.Infof("Created DeploymentConfig %v.\n", request.Name)
	default:
		return fmt.Errorf("failed to create DeploymentConfig: %s", err)
	}

	return nil
}

func (dc *DeploymentConfig) Get(namespace, fullName string) (*v1.DeploymentConfig, error) {
	log.Debug("DeploymentConfig.Get()")
	return dc.clientSet.DeploymentConfigs(namespace).Get(fullName, metav1.GetOptions{})
}

func (dc *DeploymentConfig) Delete(namespace, fullName string) error {
	log.Debug("DeploymentConfig.Delete()")
	return dc.clientSet.DeploymentConfigs(namespace).Delete(fullName, &metav1.DeleteOptions{})
}

func (dc *DeploymentConfig) Instantiate(name, namespace, fullName string) (*v1.DeploymentConfig, error) {
	log.Debug("DeploymentConfig.Instantiate()")

	request := &v1.DeploymentRequest{
		TypeMeta: metav1.TypeMeta{
			Kind:       "DeploymentRequest",
			APIVersion: "v1",
		},
		Name:   fullName,
		Force:  true,
		Latest: true,
	}

	d, err := dc.clientSet.DeploymentConfigs(namespace).Instantiate(fullName, request)
	if nil == err {
		log.Infof("Instantiated Build %v", name)
	}

	return d, err
}
