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
	"github.com/openshift/client-go/route/clientset/versioned/fake"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRouteCrd(t *testing.T) {
	projectName := "demo"
	profile := "dev"
	namespace := projectName + "-" + profile
	app := "hello-world"
	clientSet := fake.NewSimpleClientset().RouteV1()
	route := newRoute(clientSet)

	url, err := route.Create(app, namespace, 8080)
	assert.Equal(t, nil, err)
	assert.Equal(t, "", url)

	r, err := route.Get(app, namespace)
	assert.Equal(t, nil, err)
	assert.Equal(t, app, r.Name)

	err = route.Delete(app, namespace)
	assert.Equal(t, nil, err)
}
