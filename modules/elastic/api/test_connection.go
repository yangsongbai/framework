/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package api

import (
	"fmt"
	"github.com/segmentio/encoding/json"
	util2 "infini.sh/agent/lib/util"
	"infini.sh/framework/core/api"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/errors"
	"infini.sh/framework/core/model"
	"infini.sh/framework/core/util"
	"infini.sh/framework/lib/fasthttp"
	"infini.sh/framework/modules/elastic/common"
	"net/http"
	"strings"
	"time"
)

type TestAPI struct {
	api.Handler
}

var testAPI = TestAPI{}

var testInited bool

func InitTestAPI() {
	if !testInited {
		api.HandleAPIMethod(api.POST, "/elasticsearch/try_connect", testAPI.HandleTestConnectionAction)
		testInited = true
	}
}

func (h TestAPI) HandleTestConnectionAction(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	var (
		freq    = fasthttp.AcquireRequest()
		fres    = fasthttp.AcquireResponse()
		resBody = map[string]interface{}{}
	)
	defer func() {
		fasthttp.ReleaseRequest(freq)
		fasthttp.ReleaseResponse(fres)
	}()
	var config = &elastic.ElasticsearchConfig{}
	err := h.DecodeJSON(req, &config)
	if err != nil {
		panic(err)
	}
	defer req.Body.Close()
	var url string
	if config.Endpoint != "" {
		url = config.Endpoint
	} else if config.Host != "" && config.Schema != "" {
		url = fmt.Sprintf("%s://%s", config.Schema, config.Host)
		config.Endpoint = url
	} else {
		resBody["error"] = fmt.Sprintf("invalid config: %v", util.MustToJSON(config))
		h.WriteJSON(w, resBody, http.StatusInternalServerError)
		return
	}

	if url == "" {
		panic(errors.Error("invalid url: " + util.MustToJSON(config)))
	}

	if !util.SuffixStr(url, "/") {
		url = fmt.Sprintf("%s/", url)
	}

	freq.SetRequestURI(url)
	freq.Header.SetMethod("GET")

	if (config.BasicAuth == nil || (config.BasicAuth != nil && config.BasicAuth.Username == "")) &&
		config.CredentialID != "" && config.CredentialID != "manual" {
		credential, err := common.GetCredential(config.CredentialID)
		if err != nil {
			panic(err)
		}
		var dv interface{}
		dv, err = credential.Decode()
		if err != nil {
			panic(err)
		}
		if auth, ok := dv.(model.BasicAuth); ok {
			config.BasicAuth = &auth
		}
	}

	if config.BasicAuth != nil && strings.TrimSpace(config.BasicAuth.Username) != "" {
		freq.SetBasicAuth(config.BasicAuth.Username, config.BasicAuth.Password)
	}

	err = client.DoTimeout(freq, fres,10*time.Second)

	if err != nil {
		panic(err)
	}

	var statusCode = fres.StatusCode()
	if statusCode>300 ||statusCode==0 {
		resBody["error"] = fmt.Sprintf("invalid status code: %d", statusCode)
		h.WriteJSON(w, resBody, 500)
		return
	}

	b := fres.Body()
	clusterInfo := &elastic.ClusterInformation{}
	err = json.Unmarshal(b, clusterInfo)
	if err != nil {
		panic(err)
	}

	resBody["version"] = clusterInfo.Version.Number
	resBody["cluster_uuid"] = clusterInfo.ClusterUUID
	resBody["cluster_name"] = clusterInfo.ClusterName
	resBody["distribution"] = clusterInfo.Version.Distribution

	//fetch cluster health info
	freq.SetRequestURI(fmt.Sprintf("%s/_cluster/health", config.Endpoint))
	fres.Reset()
	err = client.Do(freq, fres)
	if err != nil {
		resBody["error"] = fmt.Sprintf("error on get cluster health: %v", err)
		h.WriteJSON(w, resBody, http.StatusInternalServerError)
		return
	}

	healthInfo := &elastic.ClusterHealth{}
	err = json.Unmarshal(fres.Body(), &healthInfo)
	if err != nil {
		resBody["error"] = fmt.Sprintf("error on decode cluster health info : %v", err)
		h.WriteJSON(w, resBody, http.StatusInternalServerError)
		return
	}
	resBody["status"] = healthInfo.Status
	resBody["number_of_nodes"] = healthInfo.NumberOfNodes
	resBody["number_of_data_nodes"] = healthInfo.NumberOf_data_nodes
	resBody["active_shards"] = healthInfo.ActiveShards

	//fetch local node's info
	nodeID, nodeInfo, err := util2.GetLocalNodeInfo(config.GetAnyEndpoint(), config.BasicAuth)
	if err != nil {
		resBody["error"] = fmt.Sprintf("error on decode cluster health info : %v", err)
		h.WriteJSON(w, resBody, http.StatusInternalServerError)
		return
	}

	resBody["status"] = healthInfo.Status
	resBody["number_of_nodes"] = healthInfo.NumberOfNodes
	resBody["number_of_data_nodes"] = healthInfo.NumberOf_data_nodes
	resBody["active_shards"] = healthInfo.ActiveShards
	resBody["node_uuid"] = nodeID
	resBody["node_info"] = nodeInfo

	h.WriteJSON(w, resBody, http.StatusOK)

}
