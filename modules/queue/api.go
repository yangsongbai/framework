/*
Copyright 2016 Medcl (m AT medcl.net)

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

package queue

import (
	"infini.sh/framework/core/api"
	"infini.sh/framework/core/queue"
	"infini.sh/framework/core/util"
	"net/http"
)

func RegisterAPI()  {
	api2:=api1{}
	//http://localhost:2900/queue/stats
	api.HandleAPIFunc("/queue/stats", api2.QueueStatsAction)
}

type api1 struct {
	api.Handler
}

// QueueStatsAction return queue stats information
func (handler api1) QueueStatsAction(w http.ResponseWriter, req *http.Request) {

	data := map[string]int64{}
	queues := queue.GetQueues()
	for _, q := range queues {
		data[q] = queue.Depth(q)
	}
	handler.WriteJSON(w, util.MapStr{
		"depth": data,
	}, 200)
}
