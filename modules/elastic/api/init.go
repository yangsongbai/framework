package api

import (
	"infini.sh/framework/core/api"
	"infini.sh/framework/core/api/rbac/enum"
	"infini.sh/framework/modules/elastic/common"
)

func Init(cfg common.ModuleConfig) {
	clusterAPI := APIHandler{Config: cfg}

	api.HandleAPIMethod(api.GET, "/elasticsearch/:id/health", clusterAPI.GetClusterHealth)
	api.HandleAPIMethod(api.GET, "/elasticsearch/:id/metrics", clusterAPI.HandleMetricsSummaryAction)
	api.HandleAPIMethod(api.GET, "/elasticsearch/:id/cluster_metrics", clusterAPI.HandleClusterMetricsAction)
	api.HandleAPIMethod(api.GET, "/elasticsearch/:id/node_metrics", clusterAPI.HandleNodeMetricsAction)
	api.HandleAPIMethod(api.GET, "/elasticsearch/:id/index_metrics", clusterAPI.HandleIndexMetricsAction)
	api.HandleAPIMethod(api.GET, "/elasticsearch/:id/queue_metrics", clusterAPI.HandleQueueMetricsAction)
	api.HandleAPIMethod(api.GET, "/elasticsearch/:id/storage_metrics", clusterAPI.HandleGetStorageMetricAction)

	api.HandleAPIMethod(api.POST, "/elasticsearch/", clusterAPI.RequirePermission(clusterAPI.HandleCreateClusterAction, enum.PermissionElasticsearchClusterWrite))
	api.HandleAPIMethod(api.GET, "/elasticsearch/indices", clusterAPI.ListIndex)
	api.HandleAPIMethod(api.GET, "/elasticsearch/status", clusterAPI.GetClusterStatusAction)
	api.HandleAPIMethod(api.GET, "/elasticsearch/:id", clusterAPI.RequirePermission(clusterAPI.HandleGetClusterAction, enum.PermissionElasticsearchClusterRead))
	//api.HandleAPIMethod(api.GET, "/elasticsearch/:id/nodes/kv", clusterAPI.HandleGetNodesAction)
	api.HandleAPIMethod(api.PUT, "/elasticsearch/:id", clusterAPI.RequirePermission(clusterAPI.HandleUpdateClusterAction, enum.PermissionElasticsearchClusterWrite))
	api.HandleAPIMethod(api.DELETE, "/elasticsearch/:id", clusterAPI.RequirePermission(clusterAPI.HandleDeleteClusterAction, enum.PermissionElasticsearchClusterWrite))
	api.HandleAPIMethod(api.GET, "/elasticsearch/_search", clusterAPI.RequirePermission(clusterAPI.HandleSearchClusterAction, enum.PermissionElasticsearchClusterRead))
	api.HandleAPIMethod(api.POST, "/elasticsearch/_search", clusterAPI.RequirePermission(clusterAPI.HandleSearchClusterAction, enum.PermissionElasticsearchClusterRead))

	api.HandleAPIMethod(api.POST, "/elasticsearch/:id/search_template", clusterAPI.HandleCreateSearchTemplateAction)
	api.HandleAPIMethod(api.PUT, "/elasticsearch/:id/search_template/:template_id", clusterAPI.HandleUpdateSearchTemplateAction)
	api.HandleAPIMethod(api.DELETE, "/elasticsearch/:id/search_template/:template_id", clusterAPI.HandleDeleteSearchTemplateAction)
	api.HandleAPIMethod(api.GET, "/elasticsearch/:id/search_template", clusterAPI.HandleSearchSearchTemplateAction)
	api.HandleAPIMethod(api.GET, "/elasticsearch/:id/search_template/:template_id", clusterAPI.HandleGetSearchTemplateAction)
	api.HandleAPIMethod(api.GET, "/elasticsearch/:id/search_template_history/_search", clusterAPI.HandleSearchSearchTemplateHistoryAction)
	api.HandleAPIMethod(api.POST, "/elasticsearch/:id/_render/template", clusterAPI.HandleRenderTemplateAction)
	api.HandleAPIMethod(api.POST, "/elasticsearch/:id/_search/template", clusterAPI.HandleSearchTemplateAction)

	api.HandleAPIMethod(api.POST, "/elasticsearch/:id/alias", clusterAPI.HandleAliasAction)
	api.HandleAPIMethod(api.GET, "/elasticsearch/:id/alias", clusterAPI.HandleGetAliasAction)

	api.HandleAPIMethod(api.POST, "/elasticsearch/:id/saved_objects/view", clusterAPI.RequirePermission(clusterAPI.HandleCreateViewAction, enum.PermissionViewWrite))
	api.HandleAPIMethod(api.GET, "/elasticsearch/:id/saved_objects/_find", clusterAPI.RequirePermission(clusterAPI.HandleGetViewListAction, enum.PermissionViewRead))
	api.HandleAPIMethod(api.DELETE, "/elasticsearch/:id/saved_objects/view/:view_id", clusterAPI.RequirePermission(clusterAPI.HandleDeleteViewAction, enum.PermissionViewWrite))
	api.HandleAPIMethod(api.PUT, "/elasticsearch/:id/saved_objects/view/:view_id", clusterAPI.RequirePermission(clusterAPI.HandleUpdateViewAction, enum.PermissionViewWrite))
	api.HandleAPIMethod(api.GET, "/elasticsearch/:id/internal/view-management/resolve_index/:wild", clusterAPI.HandleResolveIndexAction)
	api.HandleAPIMethod(api.POST, "/elasticsearch/:id/saved_objects/_bulk_get", clusterAPI.RequirePermission(clusterAPI.HandleBulkGetViewAction, enum.PermissionViewRead))
	api.HandleAPIMethod(api.GET, "/elasticsearch/:id/view/_fields_for_wildcard", clusterAPI.HandleGetFieldCapsAction)

	api.HandleAPIMethod(api.POST, "/elasticsearch/:id/search/ese", clusterAPI.HandleEseSearchAction)
	api.HandleAPIMethod(api.GET, "/elasticsearch/:id/search/trace_id", clusterAPI.HandleTraceIDSearchAction)
	api.HandleAPIMethod(api.POST, "/elasticsearch/:id/suggestions/values/:index", clusterAPI.HandleValueSuggestionAction)
	api.HandleAPIMethod(api.POST, "/elasticsearch/:id/setting", clusterAPI.HandleSettingAction)
	api.HandleAPIMethod(api.GET, "/elasticsearch/:id/setting/:key", clusterAPI.HandleGetSettingAction)

	api.HandleAPIMethod(api.POST, "/elasticsearch/:id/_proxy", clusterAPI.HandleProxyAction)
	api.HandleAPIMethod(api.POST, "/elasticsearch/try_connect", clusterAPI.HandleTestConnectionAction)


	api.HandleAPIMethod(api.GET, "/elasticsearch/metadata", clusterAPI.GetMetadata)
	api.HandleAPIMethod(api.GET, "/elasticsearch/hosts", clusterAPI.GetHosts)

	api.HandleAPIMethod(api.POST, "/elasticsearch/cluster/_search", clusterAPI.RequirePermission(clusterAPI.SearchClusterMetadata, enum.PermissionElasticsearchClusterRead))
	api.HandleAPIMethod(api.POST, "/elasticsearch/cluster/info", clusterAPI.RequirePermission(clusterAPI.FetchClusterInfo, enum.PermissionElasticsearchMetricRead))

	api.HandleAPIMethod(api.GET, "/elasticsearch/:id/overview/treemap", clusterAPI.RequirePermission(clusterAPI.ClusterOverTreeMap, enum.PermissionElasticsearchMetricRead))
	api.HandleAPIMethod(api.GET, "/elasticsearch/:id/info", clusterAPI.RequirePermission(clusterAPI.GetClusterInfo, enum.PermissionElasticsearchMetricRead))
	api.HandleAPIMethod(api.POST, "/elasticsearch/node/_search", clusterAPI.RequirePermission(clusterAPI.SearchNodeMetadata, enum.PermissionElasticsearchNodeRead))
	api.HandleAPIMethod(api.GET, "/elasticsearch/:id/nodes", clusterAPI.RequirePermission(clusterAPI.GetClusterNodes, enum.PermissionElasticsearchMetricRead, enum.PermissionElasticsearchNodeRead))
	api.HandleAPIMethod(api.GET, "/elasticsearch/:id/nodes/realtime", clusterAPI.RequirePermission(clusterAPI.GetRealtimeClusterNodes, enum.PermissionElasticsearchMetricRead))
	api.HandleAPIMethod(api.POST, "/elasticsearch/node/info", clusterAPI.RequirePermission(clusterAPI.FetchNodeInfo, enum.PermissionElasticsearchMetricRead))
	api.HandleAPIMethod(api.GET, "/elasticsearch/:id/indices", clusterAPI.RequirePermission(clusterAPI.GetClusterIndices, enum.PermissionElasticsearchMetricRead, enum.PermissionElasticsearchIndexRead))
	api.HandleAPIMethod(api.GET, "/elasticsearch/:id/indices/realtime", clusterAPI.RequirePermission(clusterAPI.GetRealtimeClusterIndices, enum.PermissionElasticsearchMetricRead))
	api.HandleAPIMethod(api.GET, "/elasticsearch/:id/node/:node_id/info", clusterAPI.RequirePermission(clusterAPI.GetNodeInfo, enum.PermissionElasticsearchMetricRead, enum.PermissionElasticsearchNodeRead))
	api.HandleAPIMethod(api.GET, "/elasticsearch/:id/node/:node_id/metrics", clusterAPI.RequirePermission(clusterAPI.GetSingleNodeMetrics, enum.PermissionElasticsearchMetricRead))
	api.HandleAPIMethod(api.GET, "/elasticsearch/:id/node/:node_id/indices", clusterAPI.RequirePermission(clusterAPI.getNodeIndices, enum.PermissionElasticsearchMetricRead, enum.PermissionElasticsearchIndexRead))
	api.HandleAPIMethod(api.POST, "/elasticsearch/index/_search", clusterAPI.RequirePermission(clusterAPI.SearchIndexMetadata, enum.PermissionElasticsearchIndexRead))
	api.HandleAPIMethod(api.GET, "/elasticsearch/:id/index/:index/metrics", clusterAPI.RequirePermission(clusterAPI.GetSingleIndexMetrics, enum.PermissionElasticsearchMetricRead))
	api.HandleAPIMethod(api.GET, "/elasticsearch/:id/index/:index/info", clusterAPI.RequirePermission(clusterAPI.GetIndexInfo,enum.PermissionElasticsearchIndexRead, enum.PermissionElasticsearchMetricRead))
	api.HandleAPIMethod(api.GET, "/elasticsearch/:id/index/:index/nodes", clusterAPI.RequirePermission(clusterAPI.getIndexNodes, enum.PermissionElasticsearchMetricRead, enum.PermissionElasticsearchNodeRead))
	api.HandleAPIMethod(api.POST, "/elasticsearch/index/info", clusterAPI.RequirePermission(clusterAPI.FetchIndexInfo, enum.PermissionElasticsearchMetricRead))

	api.HandleAPIMethod(api.GET, "/elasticsearch/:id/trace_template", clusterAPI.HandleSearchTraceTemplateAction)
	api.HandleAPIMethod(api.GET, "/elasticsearch/:id/trace_template/:template_id", clusterAPI.HandleGetTraceTemplateAction)
	api.HandleAPIMethod(api.POST, "/elasticsearch/:id/trace_template", clusterAPI.HandleCrateTraceTemplateAction)
	api.HandleAPIMethod(api.PUT, "/elasticsearch/:id/trace_template/:template_id", clusterAPI.HandleSaveTraceTemplateAction)
	api.HandleAPIMethod(api.DELETE, "/elasticsearch/:id/trace_template/:template_id", clusterAPI.HandleDeleteTraceTemplateAction)

	api.HandleAPIMethod(api.DELETE, "/_framework/api/local_state/elasticsearch/:type/:target", clusterAPI.HandleRemoveLocalState)
	api.HandleAPIMethod(api.POST, "/elasticsearch/activity/_search", clusterAPI.RequirePermission(clusterAPI.HandleSearchActivityAction, enum.PermissionActivityRead))
}
