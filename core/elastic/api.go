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

package elastic

import (
	"bytes"
)

type API interface {
	ScrollAPI
	MappingAPI
	TemplateAPI

	InitDefaultTemplate(templateName,indexPrefix string)

	GetMajorVersion() int

	ClusterHealth() *ClusterHealth

	ClusterVersion() string

	CreateIndex(name string, settings map[string]interface{}) error

	Index(indexName string, id interface{}, data interface{}) (*InsertResponse, error)

	Bulk(data *bytes.Buffer)

	Get(indexName, id string) (*GetResponse, error)
	Delete(indexName, id string) (*DeleteResponse, error)
	Count(indexName string) (*CountResponse, error)
	Search(indexName string, query *SearchRequest) (*SearchResponse, error)
	SearchWithRawQueryDSL(indexName string, queryDSL []byte) (*SearchResponse, error)

	GetIndexSettings(indexNames string) (*Indexes, error)
	UpdateIndexSettings(indexName string, settings map[string]interface{}) error

	IndexExists(indexName string) (bool, error)
	DeleteIndex(name string) error

	Refresh(name string) (err error)

	GetNodes() (*map[string]NodesInfo, error)
	GetIndices() (*map[string]IndexInfo, error)
	GetPrimaryShards() (*map[string]ShardInfo, error)

	SearchTasksByIds(ids []string) (*SearchResponse, error)
	Reindex(body []byte) (*ReindexResponse, error)
	DeleteByQuery(indexName string, body []byte) (*DeleteByQueryResponse, error)
}

type NodesInfo struct {
	Name    string `json:"name,omitempty"`
	Version string `json:"version,omitempty"`
	Http    struct {
		PublishAddress          string `json:"publish_address,omitempty"`
		MaxContentLengthInBytes int    `json:"max_content_length_in_bytes,omitempty"`
	} `json:"http,omitempty"`

	TotalIndexingBuffer int                    `json:"total_indexing_buffer,omitempty"`
	Attributes          map[string]interface{} `json:"attributes,omitempty"`
	Roles               []string               `json:"roles,omitempty"`
	//TODO return more nodes level settings, for later check and usage
}

type IndexInfo struct {
	ID        string `json:"id,omitempty"`
	Index        string `json:"index,omitempty"`
	Status       string `json:"status,omitempty"`
	Health       string `json:"health,omitempty"`
	Shards       int    `json:"shards,omitempty"`
	Replicas     int    `json:"replicas,omitempty"`
	DocsCount    int64  `json:"docs_count,omitempty"`
	DocsDeleted  int64  `json:"docs_deleted,omitempty"`
	StoreSize    string `json:"store_size,omitempty"`
	PriStoreSize string `json:"pri_store_size,omitempty"`
}

type ShardInfo struct {
	Index            string `json:"index,omitempty"`
	ShardID          string `json:"shard_id,omitempty"`
	Primary          bool   `json:"primary,omitempty"`
	State            string `json:"state,omitempty"`
	UnassignedReason string `json:"unassigned_reason,omitempty"`
	Docs             int64  `json:"docs_count,omitempty"`
	Store            string `json:"store_size,omitempty"`
	NodeID           string `json:"node_id,omitempty"`
	NodeName         string `json:"node_name,omitempty"`
	NodeIP           string `json:"node_ip,omitempty"`
}

type NodesResponse struct {
	ClusterName string `json:"cluster_name,omitempty"`
	Nodes       map[string]NodesInfo
}

type TemplateAPI interface {
	TemplateExists(templateName string) (bool, error)
	PutTemplate(templateName string, template []byte) ([]byte, error)
}

type MappingAPI interface {
	GetMapping(copyAllIndexes bool, indexNames string) (string, int, *Indexes, error)
	UpdateMapping(indexName string, mappings []byte) ([]byte, error)
}

type ScrollAPI interface {
	NewScroll(indexNames string, scrollTime string, docBufferCount int, query string, slicedId, maxSlicedCount int, fields string) (interface{}, error)
	NextScroll(scrollTime string, scrollId string) (interface{}, error)
}

type CatIndicesInfo struct {
	Health       string `json:"health"`
	Index        string `json:"index"`
	UUID         string `json:"uuid"`
	Pri          string `json:"pri"`
	Rep          string `json:"rep"`
	DocsCount    string `json:"docs.count"`
	DocsDeleted  string `json:"docs.deleted"`
	StoreSize    string `json:"store.size"`
	PriStoreSize string `json:"pri.store.size"`
}

type ReindexResponse struct {
	Task string `json:"task"`
}

type DeleteByQueryResponse struct {
	Deleted int `json:"deleted"`
	Total   int `json:"total"`
}
