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

package pipeline

import (
	"encoding/json"
	log "github.com/cihub/seelog"
	. "github.com/infinitbyte/framework/core/config"
	"github.com/infinitbyte/framework/core/errors"
	"github.com/infinitbyte/framework/core/global"
	"github.com/infinitbyte/framework/core/pipeline"
	"github.com/infinitbyte/framework/core/queue"
	"github.com/infinitbyte/framework/core/stats"
	"github.com/infinitbyte/framework/core/util"
	"runtime"
	"sync"
	"time"
)

var frameworkStarted bool
var runners map[string]*PipeRunner

type PipelineFrameworkModule struct {
}

type PipeRunner struct {
	config         pipeline.PipeConfig
	l              sync.Mutex
	signalChannels []*chan bool
}

func (pipe *PipeRunner) Start(config pipeline.PipeConfig) {
	if !config.Enabled {
		log.Debugf("pipeline: %s was disabled", config.Name)
		return
	}

	pipe.l.Lock()
	defer pipe.l.Unlock()
	pipe.config = config

	numGoRoutine := config.MaxGoRoutine

	pipe.signalChannels = make([]*chan bool, numGoRoutine)
	//start fetcher
	for i := 0; i < numGoRoutine; i++ {
		log.Tracef("start pipeline, %s, instance:%v", config.Name, i)
		signalC := make(chan bool, 1)
		pipe.signalChannels[i] = &signalC
		go pipe.runPipeline(&signalC, i)

	}
	log.Infof("pipeline: %s started with %v instances", config.Name, numGoRoutine)
}

func (pipe *PipeRunner) Update(config pipeline.PipeConfig) {
	pipe.Stop()
	pipe.Start(config)
}

func (pipe *PipeRunner) Stop() {
	if !pipe.config.Enabled {
		log.Debugf("pipeline: %s was disabled", pipe.config.Name)
		return
	}
	pipe.l.Lock()
	defer pipe.l.Unlock()

	for i, item := range pipe.signalChannels {
		if item != nil {
			*item <- true
		}
		log.Debug("send exit signal to fetch channel, shard:", i)
	}
}

func (pipe *PipeRunner) decodeMessage(message []byte) pipeline.Context {
	v := pipeline.Context{}
	err := json.Unmarshal(message, &v)
	if err != nil {
		panic(err)
	}
	return v
}

func (pipe *PipeRunner) runPipeline(signal *chan bool, shard int) {

	var inputMessage []byte
	for {
		select {
		case <-*signal:
			log.Trace("pipeline:", pipe.config.Name, " exit, shard:", shard)
			return
		case inputMessage = <-queue.ReadChan(pipe.config.InputQueue):
			stats.Increment("queue."+string(pipe.config.InputQueue), "pop")

			context := pipe.decodeMessage(inputMessage)

			if global.Env().IsDebug {
				log.Trace("pipeline:", pipe.config.Name, ", shard:", shard, " , message received:", util.ToJson(context, true))
			}

			pipelineConfig := pipe.config.DefaultConfig

			//TODO dynamic load pipeline config
			//url := context.GetStringOrDefault(pipeline.CONTEXT_TASK_URL, "")
			//pipelineConfigID := context.GetStringOrDefault(pipeline.CONTEXT_TASK_PipelineConfigID, "")
			//if pipelineConfigID != "" {
			//	var err error
			//	pipelineConfig, err = pipeline.GetPipelineConfig(pipelineConfigID)
			//	log.Debug("get pipeline config,", pipelineConfig.Name, ",", url, ",", pipelineConfigID)
			//	if err != nil {
			//		panic(err)
			//	}
			//}

			pipe.execute(shard, context, pipelineConfig)
			log.Trace("pipeline:", pipe.config.Name, ", shard:", shard, " , message ", context.SequenceID, " process finished")
		}
	}
}

func (pipe *PipeRunner) execute(shard int, context pipeline.Context, pipelineConfig *pipeline.PipelineConfig) {
	var p *pipeline.Pipeline
	defer func() {
		if !global.Env().IsDebug {
			if r := recover(); r != nil {
				if r == nil {
					return
				}
				var v string
				switch r.(type) {
				case error:
					v = r.(error).Error()
				case runtime.Error:
					v = r.(runtime.Error).Error()
				case string:
					v = r.(string)
				}

				log.Error("module, pipeline:", pipe.config.Name, ", shard:", shard, ", instance:", p.GetID(), " ,joint:", p.GetCurrentJoint(), ", err: ", v, ", sequence:", context.SequenceID, ", ", util.ToJson(p.GetContext(), true))
			}
		}
	}()

	p = pipeline.NewPipelineFromConfig(pipe.config.Name, pipelineConfig, &context)
	p.Run()

	if pipe.config.ThresholdInMs > 0 {
		log.Debug("pipeline:", pipe.config.Name, ", shard:", shard, ", instance:", p.GetID(), ", sleep ", pipe.config.ThresholdInMs, "ms to control speed")
		time.Sleep(time.Duration(pipe.config.ThresholdInMs) * time.Millisecond)
		log.Debug("pipeline:", pipe.config.Name, ", shard:", shard, ", instance:", p.GetID(), ", wake up now,continue crawing")
	}
}

func (module PipelineFrameworkModule) Name() string {
	return "Pipeline"
}

func (module PipelineFrameworkModule) Start(cfg *Config) {

	if frameworkStarted {
		log.Error("pipeline framework already started, please stop it first.")
		return
	}

	var config = struct {
		Runners []pipeline.PipeConfig `config:"runners"`
	}{
	//TODO load default pipe config
	//GetDefaultPipeConfig(),
	}

	cfg.Unpack(&config)

	if global.Env().IsDebug {
		log.Debug("pipeline framework config: ", config)
	}

	runners = map[string]*PipeRunner{}
	for i, v := range config.Runners {
		if v.DefaultConfig == nil {
			panic(errors.Errorf("default pipeline config can't be null, %v, %v", i, v))
		}
		if (v.InputQueue) == "" {
			panic(errors.Errorf("input queue can't be null, %v, %v", i, v))
		}

		p := &PipeRunner{config: v}
		runners[v.Name] = p
	}

	log.Debug("starting up pipeline framework")
	for _, v := range runners {
		v.Start(v.config)
	}

	frameworkStarted = true
}

func (module PipelineFrameworkModule) Stop() error {
	if frameworkStarted {
		frameworkStarted = false
		log.Debug("shutting down pipeline framework")
		for _, v := range runners {
			v.Stop()
		}
	} else {
		log.Error("pipeline framework is not started")
	}

	return nil
}
