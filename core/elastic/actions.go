/* ©INFINI, All Rights Reserved.
 * mail: contact#infini.ltd */

package elastic

import (
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/rate"
	"strings"
	"time"
)

func (node *NodeAvailable) ReportFailure() bool {
	node.configLock.Lock()
	defer node.configLock.Unlock()

	if !node.available {
		return true
	}

	node.onFailure = true
	if rate.GetRateLimiter("node_failure", node.Host, 1, 1, time.Second*1).Allow() {
		log.Debugf("vote failure ticket++ for elasticsearch [%v]", node.Host)

		node.ticket++
		//if the target host is not available for 10s, mark it down
		if (node.ticket >= 10 && time.Since(node.lastSuccess)>5*time.Second) ||time.Since(node.lastSuccess)>10*time.Second{
			log.Debugf("enough failure ticket for elasticsearch [%v], mark it down", node.Host)
			node.available = false
			node.ticket = 0
			log.Infof("node [%v] is not available", node.Host)
			return true
		}
	}
	return false
}

func (node *NodeAvailable) ReportSuccess() {

	node.lastSuccess=time.Now()

	if node.available {
		return
	}

	node.configLock.Lock()
	defer node.configLock.Unlock()

	if node.onFailure && !node.available {
		if rate.GetRateLimiter("node_available", node.Host, 1, 1, time.Second*1).Allow() {
			log.Debugf("vote success ticket++ for elasticsearch [%v]", node.Host)
			node.onFailure = false
			node.available = true
			node.ticket = 0
			log.Infof("node [%v] is available", node.Host)
		}
	}
}

func (node *NodeAvailable) IsAvailable() bool {
		node.configLock.RLock()
	defer node.configLock.RUnlock()

	return node.available
}

func (meta *ElasticsearchMetadata) IsAvailable() bool {
	if !meta.Config.Enabled {
		return false
	}

	meta.configLock.RLock()
	defer meta.configLock.RUnlock()

	return meta.clusterAvailable
}

func (meta *ElasticsearchMetadata) Init(health bool){
	meta.clusterAvailable = health
	meta.clusterOnFailure = !health
	meta.lastSuccess=time.Now()
	meta.clusterFailureTicket = 0
}

func (meta *BulkActionMetadata)GetItem() *BulkIndexMetadata {
	if meta.Index!=nil{
		return meta.Index
	}else if meta.Delete!=nil{
		return meta.Delete
	}else if meta.Create!=nil{
		return meta.Create
	}else{
		return meta.Update
	}
}

func (meta *ElasticsearchMetadata) GetPrimaryShardInfo(index string, shardID int) *ShardInfo {
	indexMap, ok := meta.PrimaryShards[index]
	if ok {
		shardInfo, ok := indexMap[shardID]
		if ok {
			return &shardInfo
		}
	}
	return nil
}

func (meta *ElasticsearchMetadata) GetActiveNodeInfo() *NodesInfo {
	for _, v := range meta.Nodes {
		return &v
	}
	return nil
}

func (meta *ElasticsearchMetadata) GetNodeInfo(nodeID string) *NodesInfo {
	info, ok := meta.Nodes[nodeID]
	if ok {
		return &info
	}
	return nil
}

func (meta *ElasticsearchMetadata) GetActiveEndpoint() string {
	return fmt.Sprintf("%s://%s",meta.GetSchema(),meta.GetActiveHost())
}

func (meta *ElasticsearchMetadata) GetActiveHost() string {
	hosts:=meta.GetSeedHosts()
	for _,v:=range hosts{
		if IsHostAvailable(v){
			return v
		}
	}
	if rate.GetRateLimiter("cluster_available", meta.Config.Name, 1, 1, time.Second*10).Allow() {
		log.Error("no host available, choose the first one, ",hosts[0])
	}
	meta.ReportFailure()
	return hosts[0]
}

func (meta *ElasticsearchMetadata) IsTLS() bool {
	return meta.GetSchema()=="https"
}

func (meta *ElasticsearchMetadata) GetSchema() string {
	if meta.Config.Schema!=""{
		return meta.Config.Schema
	}
	if meta.Config.Endpoint!=""{
		if strings.Contains(meta.Config.Endpoint, "https") {
			meta.Config.Schema= "https"
		} else {
			meta.Config.Schema= "http"
		}
		return meta.Config.Schema
	}
	if len(meta.Config.Endpoints)>0{
		for _,v:=range meta.Config.Endpoints{
			if strings.Contains(v, "https") {
				meta.Config.Schema= "https"
			} else {
				meta.Config.Schema= "http"
			}
			return meta.Config.Schema
		}
	}

	if meta.Config.Schema==""{
		meta.Config.Schema="http"
	}

	return meta.Config.Schema
}


func (meta *ElasticsearchMetadata) ReportFailure() bool {
	meta.configLock.Lock()
	defer meta.configLock.Unlock()

	if !meta.clusterAvailable {
		return true
	}

	meta.clusterOnFailure = true
	if rate.GetRateLimiter("cluster_failure", meta.Config.Name, 1, 1, time.Second*1).Allow() {
		log.Debugf("vote failure ticket++ for elasticsearch [%v]", meta.Config.Name)

		meta.clusterFailureTicket++
		//if the target host is not available for 10s, mark it down
		if (meta.clusterFailureTicket >= 10 && time.Since(meta.lastSuccess)>5*time.Second) ||time.Since(meta.lastSuccess)>10*time.Second{
			log.Debugf("enough failure ticket for elasticsearch [%v], mark it down", meta.Config.Name)
			meta.clusterAvailable = false
			meta.clusterFailureTicket = 0
			log.Infof("elasticsearch [%v] is not available", meta.Config.Name)
			return true
		}
	}
	return false
}

func (meta *ElasticsearchMetadata) ReportSuccess() {

	meta.lastSuccess=time.Now()

	if meta.clusterAvailable {
		return
	}

	meta.configLock.Lock()
	defer meta.configLock.Unlock()

	if meta.clusterOnFailure && !meta.clusterAvailable {
		if rate.GetRateLimiter("cluster_available", meta.Config.Name, 1, 1, time.Second*1).Allow() {
			log.Debugf("vote success ticket++ for elasticsearch [%v]", meta.Config.Name)
			meta.clusterOnFailure = false
			meta.clusterAvailable = true
			meta.clusterFailureTicket = 0
			log.Infof("elasticsearch [%v] is available", meta.Config.Name)
		}
	}
}
