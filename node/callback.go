package main

import (
	"encoding/json"
	"log"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
)

// func OnTaskDoneStreamHandler(h host.Host) func(s network.Stream) {
// 	return func(s network.Stream) {
// 		log.Printf("Received a stream from %s\n", s.Conn().RemotePeer())
// 		var taskCb taskCallback
// 		if err := json.NewDecoder(s).Decode(&taskCb); err != nil {
// 			log.Printf("Failed to decode task callback: %s\n", err)
// 			return
// 		}
// 		log.Printf("Received task callback: %s\n", taskCb)
// 		task, ok := taskList[taskCb.TaskID]
// 		if !ok {
// 			log.Println("Task not found")
// 			return
// 		}
// 		job, ok := jobList[task.JobID]
// 		if !ok {
// 			log.Println("Job not found")
// 			return
// 		}
// 		if taskCb.Error == nil {
// 			job.State = "completed"
// 			log.Printf("Task %s completed, result saved at %s\n", task.TaskID, task.ResultPath)
// 		} else {
// 			log.Printf("Task %s failed: %s\n", task.TaskID, task.Error)
// 		}
// 	}
// }

// func OnTaskReceivedStreamHandler(h host.Host) func(s network.Stream) {
// 	return func(s network.Stream) {
// 		log.Printf("Received a stream from %s\n", s.Conn().RemotePeer())
// 		var task taskCallback
// 		if err := json.NewDecoder(s).Decode(&task); err != nil {
// 			log.Printf("Failed to decode task callback: %s\n", err)
// 			return
// 		}
// 		log.Printf("Received task callback: %s\n", task)
// 		task.State = "completed"
// 		taskList[task.TaskID] = task
// 		log.Printf("Task %s received\n", task.TaskID)
// 	}
// }

func OnJobCompletedStreamHandler(h host.Host) func(s network.Stream) {
	return func(s network.Stream) {
		log.Printf("Received a stream from %s\n", s.Conn().RemotePeer())
		var job jobInfo
		if err := json.NewDecoder(s).Decode(&job); err != nil {
			log.Printf("Failed to decode job info: %s\n", err)
			return
		}
		log.Printf("Received job info: %s\n", job)
		job.State = "completed"
		jobList[job.ID] = job
		log.Printf("Job %s completed\n", job.ID)
	}
}
