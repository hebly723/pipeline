package main

import (
	"fmt"
	"pipeline/pipeline"
)

func main() {
	params := make(map[string]map[string]string)
	config := pipeline.ReadConfig(".")
	pipeline.InitParams(params, config.Machines)
	fmt.Printf("map初始化:%+v\n", params)
	pipeline.DoPipeline(config, params)
	pipeline.WaitEnter()
}
