/*
Copyright Medcl (m AT medcl.net)

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

package adapter

import (
	"github.com/segmentio/encoding/json"
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/util"
)

func GetMajorVersion(esConfig elastic.ElasticsearchConfig)(string, error)  {
	esVersion, err := ClusterVersion(&esConfig)
	if err != nil {
		return "", err
	}
	return esVersion.Version.Number, nil
}

func ClusterVersion(config *elastic.ElasticsearchConfig) (*elastic.ClusterInformation, error) {

	req := util.NewGetRequest(fmt.Sprintf("%s", config.Endpoint), nil)

	req.SetContentType(util.ContentTypeJson)

	if config.BasicAuth != nil {
		req.SetBasicAuth(config.BasicAuth.Username, config.BasicAuth.Password)
	}
	if config.HttpProxy != "" {
		req.SetProxy(config.HttpProxy)
	}

	response, err := util.ExecuteRequest(req)
	if err != nil {
		return nil, err
	}

	if response.StatusCode!=200{
		panic(errors.New(string(response.Body)))
	}

	version := elastic.ClusterInformation{}
	err = json.Unmarshal(response.Body, &version)
	if err != nil {
		log.Error(string(response.Body))
		return nil, err
	}
	return &version, nil
}
