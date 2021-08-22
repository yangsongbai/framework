package api

import (
	"fmt"
	log "github.com/cihub/seelog"
	"infini.sh/framework/core/api"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/orm"
	"infini.sh/framework/core/util"
	"infini.sh/framework/modules/elastic/common"
	"net/http"
	"strings"
	"time"
)

type APIHandler struct {
	api.Handler
	Config common.ModuleConfig
}

func (h *APIHandler) Client() elastic.API {
	return elastic.GetClient(h.Config.Elasticsearch)
}

func (h *APIHandler) HandleCreateClusterAction(w http.ResponseWriter, req *http.Request, ps httprouter.Params){
	var conf = &elastic.ElasticsearchConfig{}
	resBody := map[string] interface{}{
	}
	err := h.DecodeJSON(req, conf)
	if err != nil {
		resBody["error"] = err
		h.WriteJSON(w, resBody, http.StatusInternalServerError)
		return
	}
	// TODO validate data format
	esClient := elastic.GetClient(h.Config.Elasticsearch)
	id := util.GetUUID()
	conf.Created = time.Now()
	conf.Enabled=true
	conf.Updated = conf.Created
	index:=orm.GetIndexName(elastic.ElasticsearchConfig{})
	_, err = esClient.Index(index, "", id, conf)
	if err != nil {
		resBody["error"] = err
		h.WriteJSON(w, resBody, http.StatusInternalServerError)
		return
	}

	resBody["_source"] = conf
	resBody["_id"] = id
	resBody["result"] = "created"

	h.WriteJSON(w, resBody,http.StatusOK)

}

func (h *APIHandler) HandleGetClusterAction(w http.ResponseWriter, req *http.Request, ps httprouter.Params){
	resBody := map[string] interface{}{}

	id := ps.ByName("id")
	indexName := orm.GetIndexName(elastic.ElasticsearchConfig{})
	getResponse, err := h.Client().Get(indexName, "", id)
	if err != nil {
		resBody["error"] = err.Error()
		if getResponse!=nil{
			h.WriteJSON(w, resBody, getResponse.StatusCode)
		}else{
			h.WriteJSON(w, resBody, http.StatusInternalServerError)
		}
		return
	}
	h.WriteJSON(w,getResponse,200)
}

func (h *APIHandler) HandleUpdateClusterAction(w http.ResponseWriter, req *http.Request, ps httprouter.Params){
	var conf = map[string]interface{}{}
	resBody := map[string] interface{}{
	}
	err := h.DecodeJSON(req, &conf)
	if err != nil {
		resBody["error"] = err.Error()
		h.WriteJSON(w, resBody, http.StatusInternalServerError)
		return
	}
	id := ps.ByName("id")
	esClient := elastic.GetClient(h.Config.Elasticsearch)
	indexName := orm.GetIndexName(elastic.ElasticsearchConfig{})
	originConf, err := esClient.Get(indexName, "", id)
	if err != nil {
		resBody["error"] = err.Error()
		h.WriteJSON(w, resBody, http.StatusInternalServerError)
		return
	}
	source := originConf.Source
	for k, v := range conf {
		if k == "id" {
			continue
		}
		source[k] = v
	}
	conf["updated"] = time.Now()
	_, err = esClient.Index(indexName, "", id, source)
	if err != nil {
		resBody["error"] = err.Error()
		h.WriteJSON(w, resBody, http.StatusInternalServerError)
		return
	}
	resBody["_source"] = conf
	resBody["_id"] = id
	resBody["result"] = "updated"

	h.WriteJSON(w, resBody,http.StatusOK)}

func (h *APIHandler) HandleDeleteClusterAction(w http.ResponseWriter, req *http.Request, ps httprouter.Params){
	resBody := map[string] interface{}{
	}
	id := ps.ByName("id")
	esClient := elastic.GetClient(h.Config.Elasticsearch)
	response, err := esClient.Delete(orm.GetIndexName(elastic.ElasticsearchConfig{}), "", id)

	if err != nil {
		resBody["error"] = err.Error()
		if response!=nil{
			h.WriteJSON(w, resBody, response.StatusCode)
		}else{
			h.WriteJSON(w, resBody, http.StatusInternalServerError)
		}
		return
	}

	resBody["_id"] = id
	resBody["result"] = response.Result
	h.WriteJSON(w, resBody, response.StatusCode)
}

func (h *APIHandler) HandleSearchClusterAction(w http.ResponseWriter, req *http.Request, ps httprouter.Params){
	resBody := map[string] interface{}{
	}
	var (
		name = h.GetParameterOrDefault(req, "name", "")
		enabled = h.GetParameterOrDefault(req, "enabled", "")
		queryDSL = `{"query":{"bool":{"must":[%s]}}}`
		mustBuilder = &strings.Builder{}
	)
	if name != ""{
		mustBuilder.WriteString(fmt.Sprintf(`{"match":{"name": "%s"}}`, name))
	}
	if enabled != "" {
		if enabled != "true" {
			enabled = "false"
		}
		if mustBuilder.Len() > 0 {
			mustBuilder.WriteString(",")
		}
		mustBuilder.WriteString(fmt.Sprintf(`{"match":{"enabled": %s}}`, enabled))
	}

	queryDSL = fmt.Sprintf(queryDSL, mustBuilder.String())
	esClient := elastic.GetClient(h.Config.Elasticsearch)
	res, err := esClient.SearchWithRawQueryDSL(orm.GetIndexName(elastic.ElasticsearchConfig{}), []byte(queryDSL))

	if err != nil {
		resBody["error"] = err.Error()
		h.WriteJSON(w, resBody, http.StatusInternalServerError)
		return
	}

	h.WriteJSON(w, res, http.StatusOK)
}

//new
func (h *APIHandler) HandleClusterMetricsAction(w http.ResponseWriter, req *http.Request, ps httprouter.Params){
	resBody := map[string] interface{}{}
	id := ps.ByName("id")
	exists,client,err:=h.GetClusterClient(id)

	if err != nil {
		resBody["error"] = err.Error()
		h.WriteJSON(w, resBody, http.StatusInternalServerError)
		return
	}

	if !exists{
		resBody["error"] = fmt.Sprintf("cluster [%s] not found",id)
		h.WriteJSON(w, resBody, http.StatusNotFound)
		return
	}

	status:=client.GetClusterStats()

	summary:=map[string]interface{}{}
	summary["cluster_name"]=status.ClusterName
	summary["status"]=status.Status
	summary["indices_count"]=status.Indices["count"]
	summary["total_shards"]=status.Indices["shards"].(map[string]interface{})["total"]
	summary["primary_shards"]=status.Indices["shards"].(map[string]interface{})["primaries"]
	summary["replication_shards"]=status.Indices["shards"].(map[string]interface{})["replication"]
	//summary["unassigned_shards"]=status.Indices["shards"].(map[string]interface{})["primaries"]



	summary["document_count"]=status.Indices["docs"].(map[string]interface{})["count"]
	summary["deleted_document_count"]=status.Indices["docs"].(map[string]interface{})["deleted"]

	summary["used_store_bytes"]=status.Indices["store"].(map[string]interface{})["size_in_bytes"]


	summary["max_store_bytes"]=status.Nodes["fs"].(map[string]interface{})["total_in_bytes"]
	summary["available_store_bytes"]=status.Nodes["fs"].(map[string]interface{})["available_in_bytes"]


	summary["fielddata_bytes"]=status.Indices["fielddata"].(map[string]interface{})["memory_size_in_bytes"]
	summary["fielddata_evictions"]=status.Indices["fielddata"].(map[string]interface{})["evictions"]


	summary["query_cache_bytes"]=status.Indices["query_cache"].(map[string]interface{})["memory_size_in_bytes"]
	summary["query_cache_total_count"]=status.Indices["query_cache"].(map[string]interface{})["total_count"]
	summary["query_cache_hit_count"]=status.Indices["query_cache"].(map[string]interface{})["hit_count"]
	summary["query_cache_miss_count"]=status.Indices["query_cache"].(map[string]interface{})["miss_count"]
	summary["query_cache_evictions"]=status.Indices["query_cache"].(map[string]interface{})["evictions"]


	summary["segments_count"]=status.Indices["segments"].(map[string]interface{})["count"]
	summary["segments_memory_in_bytes"]=status.Indices["segments"].(map[string]interface{})["memory_in_bytes"]


	summary["nodes_count"]=status.Nodes["count"].(map[string]interface{})["total"]
	summary["version"]=status.Nodes["versions"]

	summary["mem_total_in_bytes"]=status.Nodes["os"].(map[string]interface{})["mem"].(map[string]interface{})["total_in_bytes"]
	summary["mem_used_in_bytes"]=status.Nodes["os"].(map[string]interface{})["mem"].(map[string]interface{})["used_in_bytes"]
	summary["mem_used_percent"]=status.Nodes["os"].(map[string]interface{})["mem"].(map[string]interface{})["used_percent"]


	summary["uptime"]=status.Nodes["jvm"].(map[string]interface{})["max_uptime_in_millis"]
	summary["used_jvm_bytes"]=status.Nodes["jvm"].(map[string]interface{})["mem"].(map[string]interface{})["heap_used_in_bytes"]
	summary["max_jvm_bytes"]=status.Nodes["jvm"].(map[string]interface{})["mem"].(map[string]interface{})["heap_max_in_bytes"]


	resBody["summary"] = summary

	bucketSize:=h.GetIntOrDefault(req,"bucket_size",10)//默认 10，每个 bucket 的时间范围，单位秒
	metricCount:=h.GetIntOrDefault(req,"metric_count",15*60) //默认 15分钟的区间，每分钟15个指标，也就是 15*60 个 bucket

	now := time.Now()
	//min,max are unix nanoseconds

	minStr:=h.Get(req,"min","")
	maxStr:=h.Get(req,"max","")

	var min,max int
	var rangeFrom,rangeTo time.Time
	if minStr==""{
		rangeFrom=now.Add(-time.Second*time.Duration(bucketSize*metricCount+1))
	}else{
		//try 2021-08-21T14:06:04.818Z
		rangeFrom,err=util.ParseStandardTime(minStr)
		if err!=nil{
			//try 1629637500000
			v,err:=util.ToInt(minStr)
			if err!=nil{
				log.Error("invalid timestamp:",minStr,err)
				rangeFrom=now.Add(-time.Second*time.Duration(bucketSize*metricCount+1))
			}else{
				rangeFrom=util.FromUnixTimestamp(int64(v)/1000)
			}
		}
	}

	if maxStr==""{
		rangeTo=now.Add(-time.Second*time.Duration(int(1*(float64(bucketSize)))))
	}else{
		rangeTo,err=util.ParseStandardTime(maxStr)
		if err!=nil{
			v,err:=util.ToInt(maxStr)
			if err!=nil{
				log.Error("invalid timestamp:",maxStr,err)
				rangeTo=now.Add(-time.Second*time.Duration(int(1*(float64(bucketSize)))))
			}else{
				rangeTo=util.FromUnixTimestamp(int64(v)/1000)
			}
		}
	}

	min=int(rangeFrom.UnixNano()/1e6)
	max=int(rangeTo.UnixNano()/1e6)
	hours:=rangeTo.Sub(rangeFrom).Hours()

	if hours<2{
		bucketSize=10
	}else if hours<6{
		bucketSize=30
	}else if hours<12{
		bucketSize=60
	}else if hours< 25{//1day
		bucketSize=60*5
	}else if hours<= 7*24+1{//7days
		bucketSize=60*15
	}else if hours<= 15*24+1{//15days
		bucketSize=60*30
	}else if hours< 30*24+1{ //<30 days
		bucketSize=60*60//hourly
	}else if hours<= 30*24+1{ //<30days
		bucketSize=12*60*60 //half daily
	}else if hours>= 30*24+1{ //>30days
		bucketSize=60*60*24//daily bucket
	}

	//fmt.Println(min," vs ",max,",",rangeFrom,rangeTo,"range hours:",hours)


	metrics:=h.GetClusterMetrics(id,bucketSize,min,max)
	resBody["metrics"] = metrics

	err=h.WriteJSON(w, resBody,http.StatusOK)
	if err!=nil{
		log.Error(err)
	}



}


//TODO, use expired hash
var clusters = map[string]elastic.ElasticsearchConfig{}

func (h *APIHandler) GetClusterClient(id string) (bool,elastic.API,error) {

	config,ok:=clusters[id]
	if !ok{
		indexName := orm.GetIndexName(elastic.ElasticsearchConfig{})
		getResponse, err := h.Client().Get(indexName, "", id)
		if err != nil {
			return false, nil, err
		}

		bytes:=util.MustToJSONBytes(getResponse.Source)
		cfg:=elastic.ElasticsearchConfig{}
		err=util.FromJSONBytes(bytes,&cfg)
		if err != nil {
			return false, nil, err
		}

		if getResponse.StatusCode==http.StatusNotFound{
			return false, nil, err
		}

		cfg.ID=id
		clusters[id]=cfg
		config = cfg
	}

	client:=common.InitClientWithConfig(config)
	elastic.RegisterInstance(id, config, client)

	return true,client,nil
}

func (h *APIHandler) GetClusterHealth(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	resBody := map[string] interface{}{}
	id := ps.ByName("id")
	exists,client,err:=h.GetClusterClient(id)

	if err != nil {
		resBody["error"] = err.Error()
		h.WriteJSON(w, resBody, http.StatusInternalServerError)
		return
	}

	if !exists{
		resBody["error"] = fmt.Sprintf("cluster [%s] not found",id)
		h.WriteJSON(w, resBody, http.StatusNotFound)
		return
	}

	health:=client.ClusterHealth()

	h.WriteJSON(w,health,200)
}

func newMetricItem(metricKey string) *common.MetricItem  {
	metricItem:=common.MetricItem{}

	metricItem.Key=metricKey

	//axis
	metricItem.Axis=[]*common.MetricAxis{}

	//lines
	metricItem.Lines=[]*common.MetricLine{}

	return &metricItem
}


func (h *APIHandler) GetClusterMetrics(id string,bucketSize int, min, max int) map[string]*common.MetricItem {

	bucketSizeStr:=fmt.Sprintf("%vs",bucketSize)

	metricItems:=[]*common.MetricItem{}

	metricItem:=newMetricItem("cluster_throughput")
	metricItem.AddAxi("indexing","group1",common.PositionLeft,"num","0,0","0,0.[00]",5,true)
	metricItem.AddAxi("searching","group2",common.PositionRight,"num","0,0","0,0.[00]",5,false)

	metricItem.AddLine("Indexing Rate","Total Indexing","Number of documents being indexed for primary and replica shards.","group1","index_stats.total.indexing.index_total","max",bucketSizeStr,"doc/s","num","0,0.[00]","0,0.[00]",false,true)
	metricItem.AddLine("Indexing Rate","Primary Indexing","Number of documents being indexed for primary shards.","group1","index_stats.primaries.indexing.index_total","max",bucketSizeStr,"doc/s","num","0,0.[00]","0,0.[00]",false,true)
	metricItem.AddLine("Search Rate","Total Searching","Number of search requests being executed across primary and replica shards. A single search can run against multiple shards!","group2","index_stats.total.search.query_total","max",bucketSizeStr,"query/s","num","0,0.[00]","0,0.[00]",false,true)
	metricItems=append(metricItems,metricItem)



	metricItem=newMetricItem("cluster_latency")
	metricItem.AddAxi("indexing","group1",common.PositionLeft,"num","0,0","0,0.[00]",5,true)
	metricItem.AddAxi("searching","group2",common.PositionRight,"num","0,0","0,0.[00]",5,false)

	metricItem.AddLine("Indexing","Indexing Latency","Average latency for indexing documents.","group1","index_stats.total.indexing.index_time_in_millis","max",bucketSizeStr,"ms","num","0,0.[00]","0,0.[00]",false,true)
	metricItem.AddLine("Indexing","Delete Latency","Average latency for delete documents.","group1","index_stats.total.indexing.delete_time_in_millis","max",bucketSizeStr,"ms","num","0,0.[00]","0,0.[00]",false,true)
	metricItem.AddLine("Searching","Query Latency","Average latency for searching query.","group2","index_stats.total.search.query_time_in_millis","max",bucketSizeStr,"ms","num","0,0.[00]","0,0.[00]",false,true)
	metricItem.AddLine("Searching","Fetch Latency","Average latency for searching fetch.","group2","index_stats.total.search.fetch_time_in_millis","max",bucketSizeStr,"ms","num","0,0.[00]","0,0.[00]",false,true)
	metricItem.AddLine("Searching","Scroll Latency","Average latency for searching fetch.","group2","index_stats.total.search.scroll_time_in_millis","max",bucketSizeStr,"ms","num","0,0.[00]","0,0.[00]",false,true)
	metricItems=append(metricItems,metricItem)


	metricItem=newMetricItem("cluster_storage")
	metricItem.AddAxi("storage","group1",common.PositionLeft,"bytes","0.[0]","0.[0]",5,true)
	metricItem.AddAxi("shards","group2",common.PositionRight,"num","0,0","0,0.[00]",5,false)

	metricItem.AddLine("Disk","Indices Storage","","group1","cluster_stats.indices.store.size_in_bytes","max",bucketSizeStr,"","bytes","0,0.[00]","0,0.[00]",false,false)
	metricItem.AddLine("Disk","Available Disk","","group5","cluster_stats.nodes.fs.available_in_bytes","max",bucketSizeStr,"","bytes","0,0.[00]","0,0.[00]",false,false)

	metricItem.AddLine("Indices Count","Indices Count","","group4","cluster_stats.indices.count","max",bucketSizeStr,"","num","0,0.[00]","0,0.[00]",false,false)
	metricItem.AddLine("Shards Count","Shards Count","","group3","cluster_stats.indices.shards.total","max",bucketSizeStr,"","num","0,0.[00]","0,0.[00]",false,false)
	metricItem.AddLine("Documents Count","Documents Count","","group2","cluster_stats.indices.docs.count","max",bucketSizeStr,"","num","0,0.[00]","0,0.[00]",false,false)
	metricItems=append(metricItems,metricItem)


	metricItem=newMetricItem("system_load")
	metricItem.AddAxi("load","group6",common.PositionLeft,"ratio","0.[0]","0.[0]",5,true)

	metricItem.AddLine("Load","System Load","","group6","cluster_stats.nodes.process.cpu.percent","max",bucketSizeStr,"%","ratio","0.[00]","0.[00]",false,false)
	metricItems=append(metricItems,metricItem)

	metricItem=newMetricItem("system_memory")
	metricItem.AddAxi("memory","group1",common.PositionLeft,"bytes","0.[0]","0.[0]",5,true)
	metricItem.AddAxi("memory","group2",common.PositionRight,"ratio","0.[0]","0.[0]",5,false)

	metricItem.AddLine("JVM Max Heap","Max Heap","","group1","cluster_stats.nodes.jvm.mem.heap_max_in_bytes","max",bucketSizeStr,"","bytes","0.[00]","0.[00]",false,false)
	metricItem.AddLine("JVM Used Heap","Used Heap","","group1","cluster_stats.nodes.jvm.mem.heap_used_in_bytes","max",bucketSizeStr,"","bytes","0.[00]","0.[00]",false,false)
	metricItem.AddLine("OS Used Percent","OS Used Percent","","group2","cluster_stats.nodes.os.mem.used_percent","max",bucketSizeStr,"%","ratio","0.[00]","0.[00]",false,false)
	metricItems=append(metricItems,metricItem)


	query:=map[string]interface{}{}
	query["query"]=util.MapStr{
		"bool": util.MapStr{
			"must": []util.MapStr{
				{
					"term":util.MapStr{
						"elasticsearch":util.MapStr{
							"value": id,
						},
					},
				},
			},
			"filter": []util.MapStr{
				{
					"range": util.MapStr{
						"timestamp": util.MapStr{
							"gte": min,
							"lte": max,
						},
					},
				},
			},
		},
	}

	metricData:=map[string][][]interface{}{}

	aggs:=map[string]interface{}{}

	for _,metricItem:=range metricItems{
		for _,line:=range metricItem.Lines{

			metricData[line.Metric.DataKey]=[][]interface{}{}

			aggs[line.Metric.ID]=util.MapStr{
				"max":util.MapStr{
					"field": line.Metric.Field,
				},
			}

			if line.Metric.IsDerivative{
				//add which metric keys to extract
				aggs[line.Metric.ID+"_deriv"]=util.MapStr{
					"derivative":util.MapStr{
						"buckets_path": line.Metric.ID,
					},
				}
			}
		}
	}

	query["size"]=0
	query["aggs"]= util.MapStr{
		"dates": util.MapStr{
			"date_histogram":util.MapStr{
				"field": "timestamp",
				"fixed_interval": bucketSizeStr,
			},
			"aggs":aggs,
		},
	}

	response,err:=elastic.GetClient(h.Config.Elasticsearch).SearchWithRawQueryDSL(orm.GetIndexName(common.MonitoringItem{}),util.MustToJSONBytes(query))
	if err!=nil{
		log.Error(err)
		panic(err)
	}


	var minDate,maxDate int64
	if response.StatusCode==200{
		for _,v:=range response.Aggregations{
			for _,bucket:=range v.Buckets {
				v,ok:=bucket["key"].(float64)
				if !ok{
					panic("invalid bucket key")
				}
				dateTime:=(int64(v))
				minDate=util.MinInt64(minDate,dateTime)
				maxDate=util.MaxInt64(maxDate,dateTime)
				for mk1,mv1:=range metricData{
					v1,ok:=bucket[mk1]
					if ok{
						v2,ok:=v1.(map[string]interface{})
						if ok{
							v3,ok:=v2["value"].(float64)
							if ok{
								if strings.HasSuffix(mk1, "_deriv"){
									v3 = v3/float64(bucketSize)
								}
								//only keep positive value
								if v3<0{
									continue
								}
								//v4:=int64(v3)/int64(bucketSize)
								points:=[]interface{}{dateTime,v3}
								metricData[mk1]=append(mv1,points)
							}
						}
					}
				}
			}
		}
	}

	result:=map[string]*common.MetricItem{}

	for _,metricItem:=range metricItems {
		for _,line:=range metricItem.Lines{
			line.TimeRange=common.TimeRange{Min: minDate,Max: maxDate}
			line.Data=metricData[line.Metric.DataKey]
		}
		result[metricItem.Key]=metricItem
	}

	return result
}
