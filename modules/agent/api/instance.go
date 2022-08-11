/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package api

import (
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/agent"
	"infini.sh/framework/core/api"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type APIHandler struct {
	api.Handler
}

func (h *APIHandler) heartbeat(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.MustGetParameter("instance_id")
	sm := agent.GetStateManager()
	inst, err := sm.GetAgent(id)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		log.Error(err)
		return
	}
	syncToES := inst.Status != "online"
	inst.Status = "online"
	ag, err := sm.UpdateAgent(inst, syncToES)
	if err != nil {
		log.Error(err)
	}
	taskState := map[string]string{}
	for _, cluster := range ag.Clusters {
		taskState[cluster.ClusterID] = sm.GetState(cluster.ClusterID).ClusterMetricTask.NodeUUID
	}

	h.WriteJSON(w, util.MapStr{
		"agent_id":   id,
		"result": "ok",
		"task_state": taskState,
		"timestamp": time.Now().Unix(),
	}, 200)
}

func (h *APIHandler) getIP(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	remoteHost := util.ClientIP(req)
	h.WriteJSON(w, util.MapStr{
		"ip": remoteHost,
	}, http.StatusOK)
}

func (h *APIHandler) createInstance(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	var obj = &agent.Instance{
	}
	err := h.DecodeJSON(req, obj)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		log.Error(err)
		return
	}
	if obj.Port == 0 {
		h.WriteError(w, fmt.Sprintf("invalid port [%d] of agent", obj.Port), http.StatusInternalServerError)
		return
	}
	if obj.Schema == "" {
		obj.Schema = "http"
	}
	q := &orm.Query{
		Size: 2,
	}
	remoteIP := util.ClientIP(req)
	q.Conds = orm.And(orm.Eq("host", remoteIP))
	err, result := orm.Search(obj, q)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		log.Error(err)
		return
	}
	if len(result.Result) > 0 {
		errMsg := fmt.Sprintf("agent [%s] already exists", remoteIP)
		h.WriteError(w, errMsg, http.StatusInternalServerError)
		log.Error(errMsg)
		return
	}

	//match clusters
	obj.Host = remoteIP
	clusters, err := getMatchedClusters(obj.Host, obj.Clusters)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		log.Error(err)
		return
	}

	var filterClusters []agent.ESCluster
	//remove clusters of not matched
	for i, cluster := range obj.Clusters {
		if vmap, ok := clusters[cluster.ClusterName].(map[string]interface{}); ok {
			obj.Clusters[i].ClusterID = vmap["cluster_id"].(string)
			filterClusters = append(filterClusters, obj.Clusters[i])
		}
	}
	obj.Clusters = filterClusters
	obj.Status = "online"

	err = orm.Create(obj)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		log.Error(err)
		return
	}

	sm := agent.GetStateManager()
	_, err = sm.UpdateAgent(obj, false)
	if err != nil {
		log.Error(err)
	}
	h.WriteJSON(w, util.MapStr{
		"_id":    obj.ID,
		"clusters": clusters,
		"result": "created",
	}, 200)

}

func (h *APIHandler) getInstance(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.MustGetParameter("instance_id")

	obj := agent.Instance{}
	obj.ID = id

	exists, err := orm.Get(&obj)
	if !exists || err != nil {
		h.WriteJSON(w, util.MapStr{
			"_id":   id,
			"found": false,
		}, http.StatusNotFound)
		return
	}
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		log.Error(err)
		return
	}

	h.WriteJSON(w, util.MapStr{
		"found":   true,
		"_id":     id,
		"_source": obj,
	}, 200)
}

func (h *APIHandler) updateInstance(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.MustGetParameter("instance_id")
	obj := agent.Instance{}

	obj.ID = id
	exists, err := orm.Get(&obj)
	if !exists || err != nil {
		h.WriteJSON(w, util.MapStr{
			"_id":    id,
			"result": "not_found",
		}, http.StatusNotFound)
		return
	}

	id = obj.ID
	create := obj.Created
	obj = agent.Instance{}
	err = h.DecodeJSON(req, &obj)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		log.Error(err)
		return
	}

	//protect
	obj.ID = id
	obj.Created = create
	err = orm.Update(&obj)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		log.Error(err)
		return
	}

	sm := agent.GetStateManager()
	_, err = sm.UpdateAgent(&obj, false)
	if err != nil {
		log.Error(err)
	}
	h.WriteJSON(w, util.MapStr{
		"_id":    obj.ID,
		"result": "updated",
	}, 200)
}
func (h *APIHandler) updateInstanceNodes(w http.ResponseWriter, req *http.Request, ps httprouter.Params){
	id := ps.MustGetParameter("instance_id")
	obj := agent.Instance{}
	obj.ID = id
	exists, err := orm.Get(&obj)
	if !exists || err != nil {
		h.WriteJSON(w, util.MapStr{
			"_id":    id,
			"result": "not_found",
		}, http.StatusNotFound)
		return
	}

	reqBody := []agent.ESCluster{}
	err = h.DecodeJSON(req, &reqBody)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		log.Error(err)
		return
	}
	if len(reqBody) == 0 {
		h.WriteError(w, "request body should not be empty", http.StatusInternalServerError)
		return
	}

	clusters := map[string]*agent.ESCluster{}
	var newClusters []agent.ESCluster
	for _, nc := range reqBody {
		if strings.TrimSpace(nc.ClusterID) == "" {
			newClusters = append(newClusters, nc)
			continue
		}
		clusters[nc.ClusterID] = &nc
	}
	var toUpClusters []agent.ESCluster
	for _, cluster := range obj.Clusters {
		if upCluster, ok := clusters[cluster.ClusterID]; ok {
			toUpClusters = append(toUpClusters, agent.ESCluster{
				ClusterUUID: cluster.ClusterUUID,
				ClusterName: upCluster.ClusterName,
				ClusterID: cluster.ClusterID,
				Nodes: upCluster.Nodes,
				Task: cluster.Task,
			})
			continue
		}
		//todo log delete nodes
	}
	var matchedClusters map[string]interface{}
	if len(newClusters) > 0 {
		matchedClusters, err = getMatchedClusters(obj.Host, newClusters)
		if err != nil {
			h.WriteError(w, err.Error(), http.StatusInternalServerError)
			log.Error(err)
			return
		}
		//filter already
		//for _, cluster := range toUpClusters {
		//	if _, ok := matchedClusters[cluster.ClusterName]; ok {
		//		delete(matchedClusters, cluster.ClusterName)
		//	}
		//}
	}

	for clusterName, matchedCluster := range matchedClusters {
		if vm, ok := matchedCluster.(map[string]interface{}); ok {
			toUpClusters = append(toUpClusters, agent.ESCluster{
				ClusterUUID: vm["cluster_uuid"].(string),
				ClusterName: clusterName,
				ClusterID: vm["cluster_id"].(string),
			})
		}
	}
	sm := agent.GetStateManager()
	obj.Clusters = toUpClusters
	_, err = sm.UpdateAgent(&obj, true)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		log.Error(err)
		return
	}
	resBody := util.MapStr{
		"_id":    obj.ID,
		"result": "updated",
	}
	if len(matchedClusters) > 0 {
		resBody["clusters"] = matchedClusters
	}
	h.WriteJSON(w, resBody, 200)

}
func (h *APIHandler) setTaskToInstance(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.MustGetParameter("instance_id")
	reqBody := []struct{
		ClusterID string `json:"cluster_id"`
		NodeUUID string `json:"node_uuid"`
	}{}

	err := h.DecodeJSON(req, &reqBody)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		log.Error(err)
		return
	}
	sm := agent.GetStateManager()
	for _, node := range reqBody {
		err = sm.SetAgentTask(node.ClusterID, id, node.NodeUUID)
		if err != nil {
			h.WriteError(w, err.Error(), http.StatusInternalServerError)
			log.Error(err)
			return
		}
	}

	h.WriteJSON(w, util.MapStr{
		"result": "success",
	}, 200)
}

func (h *APIHandler) deleteInstance(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	id := ps.MustGetParameter("instance_id")

	obj := agent.Instance{}
	obj.ID = id

	exists, err := orm.Get(&obj)
	if !exists || err != nil {
		h.WriteJSON(w, util.MapStr{
			"_id":    id,
			"result": "not_found",
		}, http.StatusNotFound)
		return
	}

	err = orm.Delete(&obj)
	if err != nil {
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		log.Error(err)
		return
	}
	agent.GetStateManager().DeleteAgent(obj.ID)

	h.WriteJSON(w, util.MapStr{
		"_id":    obj.ID,
		"result": "deleted",
	}, 200)
}

func (h *APIHandler) searchInstance(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {

	var (
		keyword        = h.GetParameterOrDefault(req, "keyword", "")
		queryDSL    = `{"query":{"bool":{"must":[%s]}}, "size": %d, "from": %d}`
		strSize     = h.GetParameterOrDefault(req, "size", "20")
		strFrom     = h.GetParameterOrDefault(req, "from", "0")
		mustBuilder = &strings.Builder{}
	)
	if keyword != "" {
		mustBuilder.WriteString(fmt.Sprintf(`{"query_string":{"default_field":"*","query": "%s"}}`, keyword))
	}
	size, _ := strconv.Atoi(strSize)
	if size <= 0 {
		size = 20
	}
	from, _ := strconv.Atoi(strFrom)
	if from < 0 {
		from = 0
	}

	q := orm.Query{}
	queryDSL = fmt.Sprintf(queryDSL, mustBuilder.String(), size, from)
	q.RawQuery = []byte(queryDSL)

	err, res := orm.Search(&agent.Instance{}, &q)
	if err != nil {
		log.Error(err)
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//searchRes := elastic.SearchResponse{}
	//util.MustFromJSONBytes(res.Raw, &searchRes)
	//for _, hit := range searchRes.Hits.Hits {
	//	hit.Source["task_count"] =
	//}


	h.Write(w, res.Raw)
}

func (h *APIHandler) getClusterInstance(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	clusterID := h.GetParameterOrDefault(req, "cluster_id", "")
	if clusterID == "" {
		h.WriteError(w, "parameter cluster_id should not be empty", http.StatusInternalServerError)
		return
	}
	esClient := elastic.GetClient(clusterID)
	nodes, err := esClient.CatNodes("id,ip,name")
	if err != nil {
		log.Error(err)
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	nodesM := make(map[string]*struct{
		NodeID string
		IP string
		Name string
		AgentHost string
		Owner bool
	}, len(nodes))
	for _, node := range nodes {
		nodesM[node.Id] = &struct {
			NodeID  string
			IP      string
			Name    string
			AgentHost string
			Owner   bool
		}{NodeID: node.Id, IP: node.Ip, Name: node.Name }
	}
	query := util.MapStr{
		"query": util.MapStr{
			"term": util.MapStr{
				"clusters.cluster_id": util.MapStr{
					"value": clusterID,
				},
			},
		},
	}
	q := &orm.Query{
		RawQuery: util.MustToJSONBytes(query),
	}
	err, result := orm.Search(agent.Instance{}, q)
	if err != nil {
		log.Error(err)
		h.WriteError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for _, row := range result.Result {
		buf := util.MustToJSONBytes(row)
		inst := &agent.Instance{}
		util.MustFromJSONBytes(buf, inst)
		for _, cluster := range inst.Clusters {
			for _, n := range cluster.Nodes {
				if _, ok := nodesM[n.UUID]; ok {
					nodesM[n.UUID].AgentHost = inst.Host
					nodesM[n.UUID].Owner = cluster.Task.ClusterMetric.TaskNodeID == n.UUID
				}
			}
		}
	}

	h.WriteJSON(w, nodesM, 200)
}


func getMatchedClusters(host string, clusters []agent.ESCluster) (map[string]interface{}, error){
	resultClusters := map[string] interface{}{}
	for _, cluster := range clusters {
		queryDsl := util.MapStr{
			"query": util.MapStr{
				"bool": util.MapStr{
					"should": []util.MapStr{
						{
							"term": util.MapStr{
								"cluster_uuid": util.MapStr{
									"value": cluster.ClusterUUID,
								},
							},
						},
						{
							"bool": util.MapStr{
								"minimum_should_match": 1,
								"must": []util.MapStr{
									{
										"prefix": util.MapStr{
											"host": util.MapStr{
												"value": host,
											},
										},
									},
								},
								"should": []util.MapStr{
									{
										"term": util.MapStr{
											"raw_name": util.MapStr{
												"value": cluster.ClusterName,
											},
										},
									},
									{
										"term": util.MapStr{
											"name": util.MapStr{
												"value": cluster.ClusterName,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		}
		q := &orm.Query{
			RawQuery: util.MustToJSONBytes(queryDsl),
		}
		err, result := orm.Search(elastic.ElasticsearchConfig{}, q)
		if err != nil {
			return nil, err
		}
		if len(result.Result) == 1 {
			buf := util.MustToJSONBytes(result.Result[0])
			esConfig := elastic.ElasticsearchConfig{}
			util.MustFromJSONBytes(buf, &esConfig)
			resultClusters[cluster.ClusterName] = map[string]interface{}{
				"cluster_id": esConfig.ID,
				"cluster_uuid": esConfig.ClusterUUID,
				"basic_auth": esConfig.BasicAuth,
			}
		}
	}
	return resultClusters, nil
}