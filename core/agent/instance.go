/* Copyright © INFINI Ltd. All rights reserved.
 * Web: https://infinilabs.com
 * Email: hello#infini.ltd */

package agent

import (
	"fmt"
	"time"
)

type Instance struct {
	ID      string    `json:"id,omitempty"      elastic_meta:"_id" elastic_mapping:"id: { type: keyword }"`
	Created time.Time `json:"created,omitempty" elastic_mapping:"created: { type: date }"`
	Updated time.Time `json:"updated,omitempty" elastic_mapping:"updated: { type: date }"`
	Schema      string `json:"schema,omitempty" elastic_mapping:"schema: { type: keyword }"`
	Port uint `json:"port,omitempty" elastic_mapping:"port: { type: keyword }"`
	IPS     []string               `json:"ips,omitempty" elastic_mapping:"ips: { type: keyword,copy_to:search_text }"`
	Host    string                 `json:"host" elastic_mapping:"host: { type: keyword,copy_to:search_text }"`
	Version map[string]interface{} `json:"version,omitempty" elastic_mapping:"version: { type: object }"`
	Clusters []ESCluster `json:"clusters,omitempty" elastic_mapping:"clusters: { type: object }"`
	Tags [] string `json:"tags,omitempty" elastic_mapping:"tags: { type: keyword,copy_to:search_text }"`
	Status string `json:"status,omitempty" elastic_mapping:"status: { type: keyword, copy_to:search_text }"`
	Confirmed bool `json:"confirmed" elastic_mapping:"confirmed: { type: keyword }"`
	Timestamp time.Time `json:"timestamp" elastic_mapping:"timestamp: { type: date }"`
	SearchText    string      `json:"search_text,omitempty" elastic_mapping:"search_text:{type:text,index_prefixes:{},index_phrases:true, analyzer:suggest_text_search }"`
}

func (inst *Instance) GetEndpoint() string{
	return fmt.Sprintf("%s://%s:%d", inst.Schema, inst.Host, inst.Port)
}

type ESCluster struct {
	ClusterUUID string `json:"cluster_uuid,omitempty" elastic_mapping:"cluster_uuid: { type: keyword,copy_to:search_text }"`
	ClusterID string   `json:"cluster_id,omitempty" elastic_mapping:"cluster_id: { type: keyword,copy_to:search_text }"`
	ClusterName string `json:"cluster_name,omitempty" elastic_mapping:"cluster_name: { type: keyword,copy_to:search_text }"`
	Nodes  []ESNode `json:"nodes,omitempty" elastic_mapping:"nodes: { type: object}"`
	//TaskOwner bool `json:"task_owner" elastic_mapping:"task_owner: { type: keyword }"`
	//TaskNodeID string `json:"task_node_id" elastic_mapping:"task_node_id: { type: keyword }"`
	Task Task `json:"task,omitempty" elastic_mapping:"task: { type: object}"`
	BasicAuth *BasicAuth `json:"basic_auth,omitempty"`
}
type Task struct {
	ClusterMetric ClusterMetricTask `json:"cluster_metric,omitempty" elastic_mapping:"cluster_metric: { type: object}"`
	NodeMetric *NodeMetricTask `json:"node_metric,omitempty" elastic_mapping:"node_metric: { type: object}"`
}

type ClusterMetricTask struct {
	Owner bool `json:"owner" elastic_mapping:"owner: { type: keyword }"`
	TaskNodeID string `json:"task_node_id" elastic_mapping:"task_node_id: { type: keyword }"`
}

type NodeMetricTask struct {
	Owner bool `json:"owner" elastic_mapping:"owner: { type: keyword }"`
	ExtraNodes []string `json:"extra_nodes,omitempty" elastic_mapping:"extra_nodes: { type: keyword}"`
}

type ShortState struct {
	//AgentID string
	//NodeUUID string
	ClusterMetricTask ClusterMetricTaskState

}
type ClusterMetricTaskState struct {
	AgentID string
	NodeUUID string
}

type ESNode struct {
	UUID string `json:"uuid" elastic_mapping:"uuid: { type: keyword,copy_to:search_text }"`
	Name string `json:"name" elastic_mapping:"name: { type: keyword,copy_to:search_text }"`
}

type BasicAuth struct {
	Username string `json:"username,omitempty" config:"username" elastic_mapping:"username:{type:keyword}"`
	Password string `json:"password,omitempty" config:"password" elastic_mapping:"password:{type:keyword}"`
}