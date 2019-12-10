package main

import (
	"fmt"
	"log"
	"github.com/zeebe-io/zeebe/clients/go/pkg/entities"
	"github.com/zeebe-io/zeebe/clients/go/pkg/worker"
	"github.com/zeebe-io/zeebe/clients/go/pkg/zbc"
)

const BrokerAddr = "0.0.0.0:26500"

func main() {
	client, err := zbc.NewClient(&zbc.ClientConfig{
		GatewayAddress:         BrokerAddr,
		UsePlaintextConnection: true,
	})

	if err != nil {
		panic(err)
	}

	// After the workflow is deployed.
	variables := make(map[string]interface{})
	variables["orderId"] = "31243"

	request, err := client.NewCreateInstanceCommand().BPMNProcessId("Process_Simple").LatestVersion().VariablesFromMap(variables)
	if err != nil {
		panic(err)
	}

	msg, err := request.Send()
	if err != nil {
		panic(err)
	}

	fmt.Println(msg.String())

	jobWorker := client.NewJobWorker().JobType("notifier").Handler(handleJob).Open()
	defer jobWorker.Close()

	jobWorker.AwaitClose()
}

func handleJob(client worker.JobClient, job entities.Job) {
	jobKey := job.GetKey()

	headers, err := job.GetCustomHeadersAsMap()
	if err != nil {
		// failed to handle job as we require the custom job headers
		failJob(client, job)
		return
	}

	variables, err := job.GetVariablesAsMap()
	if err != nil {
		// failed to handle job as we require the variables
		failJob(client, job)
		return
	}

	variables["totalPrice"] = 46.50
	request, err := client.NewCompleteJobCommand().JobKey(jobKey).VariablesFromMap(variables)
	if err != nil {
		// failed to set the updated variables
		failJob(client, job)
		return
	}

	log.Println("Complete job", jobKey, "of type", job.Type)
	log.Println("Processing order:", variables["orderId"])
	log.Println("Collect money using payment method:", headers["method"])

	request.Send()
}

func failJob(client worker.JobClient, job entities.Job) {
	log.Println("Failed to complete job", job.GetKey())
	client.NewFailJobCommand().JobKey(job.GetKey()).Retries(job.Retries - 1).Send()
}
