package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	httprouter "infini.sh/framework/core/api/router"
	"infini.sh/framework/core/elastic"
	"infini.sh/framework/core/util"
	"infini.sh/framework/core/orm"
	"sort"
	"strconv"
	"strings"
	"time"
)

func (h *APIHandler) HandleCreateIndexPatternAction(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	resBody := map[string]interface{}{
	}

	targetClusterID := ps.ByName("id")
	exists,_,err:=h.GetClusterClient(targetClusterID)

	if err != nil {
		resBody["error"] = err.Error()
		h.WriteJSON(w, resBody, http.StatusInternalServerError)
		return
	}

	if !exists{
		resBody["error"] = fmt.Sprintf("cluster [%s] not found",targetClusterID)
		h.WriteJSON(w, resBody, http.StatusNotFound)
		return
	}

	var indexPattern = &elastic.IndexPatternRequest{}

	err = h.DecodeJSON(req, indexPattern)
	if err != nil {
		resBody["error"] = err
		h.WriteJSON(w, resBody, http.StatusInternalServerError)
		return
	}

	esClient := elastic.GetClient(h.Config.Elasticsearch)
	id := util.GetUUID()
	indexPattern.Attributes.UpdatedAt = time.Now()
	indexPattern.Attributes.ClusterID = targetClusterID
	_, err = esClient.Index(orm.GetIndexName(indexPattern.Attributes),"", id, indexPattern.Attributes)
	if err != nil {
		resBody["error"] = err
		h.WriteJSON(w, resBody, http.StatusInternalServerError)
		return
	}
	resBody = map[string]interface{}{
		"id": id,
		"type":"index-pattern",
		"updated_at": indexPattern.Attributes.UpdatedAt,
		"attributes": indexPattern.Attributes,
		"namespaces":[]string{"default"},
	}
	h.WriteJSON(w, resBody,http.StatusOK)
}

func (h *APIHandler) HandleGetIndexPatternListAction(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	resBody := map[string]interface{}{
	}

	targetClusterID := ps.ByName("id")
	strSize := h.GetParameterOrDefault(req, "per_page", "10000")
	size, _ := strconv.Atoi(strSize)
	search := h.GetParameterOrDefault(req, "search", "")
	if search != "" {
		search = fmt.Sprintf(`,{"match":{"title":"%s"}}`, search)
	}

	esClient := elastic.GetClient(h.Config.Elasticsearch)

	queryDSL :=[]byte(fmt.Sprintf(`{"_source":["title", "updated_at"],"size": %d, "query":{"bool":{"must":{"match":{"cluster_id":"%s"}}%s}}}`, size, targetClusterID, search))

	searchRes, err := esClient.SearchWithRawQueryDSL(orm.GetIndexName(elastic.IndexPattern{}),queryDSL)
	if err != nil {
		resBody["error"] = err
		h.WriteJSON(w, resBody, http.StatusInternalServerError)
		return
	}
	var total = len(searchRes.Hits.Hits)
	if totalVal, ok := searchRes.Hits.Total.(map[string]interface{}); ok {
		total = int(totalVal["value"].(float64))
	}
	resBody =  map[string]interface{}{
		"per_page": size,
		"total": total,
	}
	var savedObjects = make([]map[string]interface{},0, len(searchRes.Hits.Hits))
	for _, hit := range searchRes.Hits.Hits {
		var savedObject = map[string]interface{}{
			"id": hit.ID,
			"attributes": map[string]interface{}{
				"title": hit.Source["title"],
			},
			"score": 0,
			"type": "index-pattern",
			"namespaces":[]string{"default"},
			"updated_at": hit.Source["updated_at"],
		}
		savedObjects = append(savedObjects, savedObject)
	}
	resBody["saved_objects"] = savedObjects
	h.WriteJSON(w, resBody,http.StatusOK)
}

func (h *APIHandler) HandleDeleteIndexPatternAction(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	resBody := map[string]interface{}{
	}

	//targetClusterID := ps.ByName("id")
	indexPatternID := ps.ByName("indexPatternID")

	esClient := elastic.GetClient(h.Config.Elasticsearch)

	_, err := esClient.Delete(orm.GetIndexName(elastic.IndexPattern{}), "", indexPatternID)
	if err != nil {
		resBody["error"] = err
		h.WriteJSON(w, resBody, http.StatusInternalServerError)
		return
	}
	h.WriteJSON(w, resBody,http.StatusOK)
}



func (h *APIHandler) HandleResolveIndexAction(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	resBody := map[string]interface{}{
	}

	targetClusterID := ps.ByName("id")
	wild := ps.ByName("wild")
	wild = strings.ReplaceAll(wild, "*", "")

	exists,client,err:=h.GetClusterClient(targetClusterID)

	if err != nil {
		resBody["error"] = err.Error()
		h.WriteJSON(w, resBody, http.StatusInternalServerError)
		return
	}

	if !exists{
		resBody["error"] = fmt.Sprintf("cluster [%s] not found",targetClusterID)
		h.WriteJSON(w, resBody, http.StatusNotFound)
		return
	}

	res, err := client.GetAliasesAndIndices()
	if err != nil {
		resBody["error"] = err
		h.WriteJSON(w, resBody, http.StatusInternalServerError)
		return
	}
	if wild != "" {
		var (
			aliases = []elastic.AAIR_Alias{}
			indices = []elastic.AAIR_Indices{}
		)
		for _, alias := range res.Aliases {
			if strings.HasPrefix(alias.Name, wild) {
				aliases = append(aliases, alias)
			}
		}
		for _, index := range res.Indices {
			if strings.HasPrefix(index.Name, wild) {
				indices = append(indices, index)
			}
		}
		res.Indices= indices
		res.Aliases = aliases
	}

	h.WriteJSON(w, res,http.StatusOK)
}

func (h *APIHandler) HandleBulkGetIndexPatternAction(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	resBody := map[string]interface{}{
	}
	targetClusterID := ps.ByName("id")
	var reqIDs = []struct {
		ID string `json:"id"`
	}{}

	err := h.DecodeJSON(req, &reqIDs)
	if err != nil {
		resBody["error"] = err
		h.WriteJSON(w, resBody, http.StatusInternalServerError)
		return
	}
	strIDs := []string{}
	for _, reqID := range reqIDs {
		strIDs = append(strIDs, fmt.Sprintf(`"%s"`, reqID.ID))
	}
	esClient := elastic.GetClient(h.Config.Elasticsearch)


	queryDSL :=[]byte(fmt.Sprintf(`{"query": {"bool": {"must": [{"terms": {"_id": [%s]}},
		{"match": {"cluster_id": "%s"}}]}}}`, strings.Join(strIDs, ","), targetClusterID))
	searchRes, err := esClient.SearchWithRawQueryDSL(orm.GetIndexName(elastic.IndexPattern{}),queryDSL)
	if err != nil {
		resBody["error"] = err
		h.WriteJSON(w, resBody, http.StatusInternalServerError)
		return
	}
	var savedObjects = make([]map[string]interface{},0, len(searchRes.Hits.Hits))
	for _, hit := range searchRes.Hits.Hits {
		var savedObject = map[string]interface{}{
			"id": hit.ID,
			"attributes": map[string]interface{}{
				"title": hit.Source["title"],
				"fields": hit.Source["fields"],
				"timeFieldName":  hit.Source["timeFieldName"],
				"fieldFormatMap":  hit.Source["fieldFormatMap"],
			},
			"score": 0,
			"type": "index-pattern",
			"namespaces":[]string{"default"},
			"migrationVersion": map[string]interface{}{"index-pattern": "7.6.0"},
			"updated_at": hit.Source["updated_at"],
		}
		savedObjects = append(savedObjects, savedObject)
	}
	resBody["saved_objects"] = savedObjects
	h.WriteJSON(w, resBody,http.StatusOK)
}

func (h *APIHandler) HandleUpdateIndexPatternAction(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	resBody := map[string]interface{}{
	}

	targetClusterID := ps.ByName("id")
	exists,_,err:=h.GetClusterClient(targetClusterID)

	if err != nil {
		resBody["error"] = err.Error()
		h.WriteJSON(w, resBody, http.StatusInternalServerError)
		return
	}

	if !exists{
		resBody["error"] = fmt.Sprintf("cluster [%s] not found",targetClusterID)
		h.WriteJSON(w, resBody, http.StatusNotFound)
		return
	}

	var indexPattern = &elastic.IndexPatternRequest{}

	err = h.DecodeJSON(req, indexPattern)
	if err != nil {
		resBody["error"] = err
		h.WriteJSON(w, resBody, http.StatusInternalServerError)
		return
	}
	if indexPattern.Attributes.Title == "" {
		resBody["error"] = "miss title"
		h.WriteJSON(w, resBody, http.StatusInternalServerError)
		return
	}
	id := ps.ByName("indexPatternID")
	esClient := elastic.GetClient(h.Config.Elasticsearch)
	indexPattern.Attributes.UpdatedAt = time.Now()
	indexPattern.Attributes.ClusterID = targetClusterID
	_, err = esClient.Index(orm.GetIndexName(indexPattern.Attributes),"", id, indexPattern.Attributes)
	if err != nil {
		resBody["error"] = err
		h.WriteJSON(w, resBody, http.StatusInternalServerError)
		return
	}
	h.WriteJSON(w, indexPattern.Attributes ,http.StatusOK)
}

func (h *APIHandler) HandleGetFieldCapsAction(w http.ResponseWriter, req *http.Request, ps httprouter.Params) {
	resBody := map[string]interface{}{
	}

	targetClusterID := ps.ByName("id")
	exists,client,err:=h.GetClusterClient(targetClusterID)

	if err != nil {
		resBody["error"] = err.Error()
		h.WriteJSON(w, resBody, http.StatusInternalServerError)
		return
	}

	if !exists{
		resBody["error"] = fmt.Sprintf("cluster [%s] not found",targetClusterID)
		h.WriteJSON(w, resBody, http.StatusNotFound)
		return
	}

	pattern := h.GetParameterOrDefault(req, "pattern", "*")
	metaFields := req.URL.Query()["meta_fields"]
	//fmt.Println(metaFields)

	buf, err := client.FieldCaps(pattern)
	if err != nil {
		resBody["error"] = err
		h.WriteJSON(w, resBody, http.StatusInternalServerError)
		return
	}
	var fieldCaps = &elastic.FieldCapsResponse{}
	err = json.Unmarshal(buf, fieldCaps)
	if err != nil {
		resBody["error"] = err
		h.WriteJSON(w, resBody, http.StatusInternalServerError)
		return
	}
	kbnFields := []KbnField{}
	for filedName, fieldCaps := range fieldCaps.Fields {
		if strings.HasPrefix(filedName, "_") && !isValidMetaField(filedName, metaFields){
			continue
		}
		var (
			typ string
			searchable bool
			aggregatable bool
			esTypes []string
			readFromDocValues bool
		)

		for esType, capsByType := range fieldCaps {
			if len(fieldCaps) > 1 {
				typ = "conflict"
			}else{
				typ = castEsToKbnFieldTypeName(esType)
			}
			esTypes = append(esTypes, esType)
			searchable = capsByType.Searchable
			aggregatable = capsByType.Aggregatable
			readFromDocValues = shouldReadFieldFromDocValues(esType, aggregatable)
		}
		if typ == "object" || typ == "nested"{
			continue
		}
		kbnFields = append(kbnFields, KbnField{
			Name: filedName,
			Aggregatable:  aggregatable,
			Type: typ,
			Searchable: searchable,
			ReadFromDocValues: readFromDocValues,
			ESTypes: esTypes,
		})
	}
	sort.Slice(kbnFields, func(i, j int)bool{
		return kbnFields[i].Name < kbnFields[j].Name
	})
	resBody["fields"] = kbnFields
	h.WriteJSON(w,resBody ,http.StatusOK)
}

func isValidMetaField(fieldName string, metaFields []string) bool {
	for _, f := range metaFields {
		if f == fieldName {
			return true
		}
	}
	return false
}

func shouldReadFieldFromDocValues(esType string, aggregatable bool) bool {
	return aggregatable && !(esType == "text" || esType == "geo_shape") && !strings.HasPrefix(esType, "_")
}

func castEsToKbnFieldTypeName(esType string) string {
	kbnTypes := createKbnFieldTypes()
	for _, ftype := range kbnTypes {
		for _, esType1 := range ftype.ESTypes {
			if esType1 == esType {
				return ftype.Name
			}
		}
	}
	return "unknown"
}


type KbnField struct {
	Aggregatable bool `json:"aggregatable"`
	ESTypes []string `json:"esTypes"`
	Name string `json:"name"`
	ReadFromDocValues bool `json:"readFromDocValues"`
	Searchable bool `json:"searchable"`
	Type string `json:"type"`
}

type KbnFieldType struct {
	Name string
	ESTypes []string
}
func createKbnFieldTypes() []KbnFieldType{
	return []KbnFieldType{
		{
			Name: "string",
			ESTypes: []string{
				"text", "keyword", "_type", "_id","_index",
			},
		},{
			Name:"number",
			ESTypes: []string{
				"float", "half_float", "scaled_float", "double","integer", "long", "unsigned_long", "short", "byte","token_count",
			},
		},{
			Name: "date",
			ESTypes: []string{
				"date", "date_nanos",
			},
		},{
			Name:"ip",
			ESTypes: []string{
				"ip",
			},
		}, {
			Name:"boolean",
			ESTypes: []string{
				"boolean",
			},
		},{
			Name:"object",
			ESTypes: []string{
				"object",
			},
		},{
			Name:"nested",
			ESTypes: []string{
				"nested",
			},
		},{
			Name:"geo_point",
			ESTypes: []string{
				"geo_point",
			},
		},{
			Name:"geo_shape",
			ESTypes: []string{
				"geo_shape",
			},
		},{
			Name:"attachment",
			ESTypes: []string{
				"attachment",
			},
		},{
			Name:"murmur3",
			ESTypes: []string{
				"murmur3",
			},
		},{
			Name:"_source",
			ESTypes: []string{
				"_source",
			},
		},{
			Name:"histogram",
			ESTypes: []string{
				"histogram",
			},
		},{
			Name:"conflict",
		},{
			Name:"unknown",
		},
	}
}
