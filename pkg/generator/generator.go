// Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package generator

import (
	"embed"
	"fmt"
	"path/filepath"

	"github.com/gardener/gardener-extension-os-gardenlinux/pkg/apis/gardenlinux"
	gardenlinuxinstall "github.com/gardener/gardener-extension-os-gardenlinux/pkg/apis/gardenlinux/install"

	controllercmd "github.com/gardener/gardener/extensions/pkg/controller/cmd"
	ostemplate "github.com/gardener/gardener/extensions/pkg/controller/operatingsystemconfig/oscommon/template"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	runtimeutils "k8s.io/apimachinery/pkg/util/runtime"
)

var (
	cmd                = "/usr/bin/env bash %s"
	cloudInitGenerator *ostemplate.CloudInitGenerator
	decoder            runtime.Decoder
)

//go:embed templates/*
var templates embed.FS

func init() {
	scheme := runtime.NewScheme()
	if err := gardenlinuxinstall.AddToScheme(scheme); err != nil {
		controllercmd.LogErrAndExit(err, "Could not update scheme")
	}
	decoder = serializer.NewCodecFactory(scheme).UniversalDecoder()

	cloudInitTemplateString, err := templates.ReadFile(filepath.Join("templates", "cloud-init.gardenlinux.template"))
	runtimeutils.Must(err)

	cloudInitTemplate, err := ostemplate.NewTemplate("cloud-init").Parse(string(cloudInitTemplateString))
	runtimeutils.Must(err)

	cloudInitGenerator = ostemplate.NewCloudInitGenerator(cloudInitTemplate, ostemplate.DefaultUnitsPath, cmd, func(osc *extensionsv1alpha1.OperatingSystemConfig) (map[string]interface{}, error) {
		if osc.Spec.Type != gardenlinux.OSTypeGardenLinux {
			return nil, nil
		}

		values := map[string]interface{}{
			"LinuxSecurityModule": "AppArmor",
		}

		if osc.Spec.ProviderConfig == nil {
			return values, nil
		}

		obj := &gardenlinux.OperatingSystemConfiguration{}
		if _, _, err := decoder.Decode(osc.Spec.ProviderConfig.Raw, nil, obj); err != nil {
			return nil, fmt.Errorf("failed to decode provider config: %+v", err)
		}

		if obj.LinuxSecurityModule != nil {
			values["LinuxSecurityModule"] = *obj.LinuxSecurityModule
		}

		return values, nil
	})
}

// CloudInitGenerator is the generator which will genereta the cloud init yaml
func CloudInitGenerator() *ostemplate.CloudInitGenerator {
	return cloudInitGenerator
}
