/*
* Copyright (c) 2017-2020. Canal+ Group
* All rights reserved
 */
package cmd

import (
	"context"
	"fmt"
	"errors"
	"strings"
	"os"
	"os/signal"
	"syscall"
	"strconv"
	"github.com/Shopify/sarama"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"rocket-storagemanager/internal/kafka"
	"rocket-storagemanager/internal/filesystem"
    "rocket-storagemanager/internal/model/config"
    "rocket-storagemanager/internal/utils"
)

var cfgFile string

// Sarama configuration options
var (
	version  = "2.1.1"
	group    = "rocket-storagemanager-testThread"
	assignor = "range"
	oldest   = true
)

var logDataDog *log.Logger

func getConfFromEnv() (config.Params, error) {
    brokerEnv := os.Getenv("KAFKA_BROKERS_SERVERS");
    nbConsumerStrEnv := os.Getenv("KAFKA_CONSUMER_NUMBER");
    relativePathEnv := os.Getenv("ROCKET_WORKSPACE_PATH");
    topicEnv := os.Getenv("KAFKA_TOPIC_NAME");
    if (brokerEnv == "" || nbConsumerStrEnv == "" || relativePathEnv == "" || topicEnv == ""){
        logDataDog.Error("ERROR: Missing one or multiple env variables, brokerEnv:", brokerEnv,
        "nbConsumerStrEnv:", nbConsumerStrEnv, "relativePathEnv:", relativePathEnv, "topicEnv:", topicEnv)
    	return config.Params{}, errors.New("Missing one or multiple env variables")
    }
    nbEnv, err := strconv.ParseInt(nbConsumerStrEnv, 10, 16)
    if err != nil {
        logDataDog.Error("ERROR: Invalid consumer number argument:", err)
        return config.Params{}, errors.New("Invalid consumer number argument")
    }
    nbConsumerIntEnv := int(nbEnv)
        if (nbConsumerIntEnv > 20) {
            logDataDog.Error("ERROR: Too many consumers, max 20")
            return config.Params{}, errors.New("Too many consumers");
    }
    paramsFromEnv := config.Params{MarkOnErrorFlag: true, MarkOnNotFoundFlag: true, BrokersUrls: brokerEnv,
        ConsumersNumber: nbConsumerIntEnv, RelativePath: relativePathEnv, Topic: topicEnv, AsCLI: false}
    return paramsFromEnv, nil
}

func getArgsAndFlags(args []string, cmd *cobra.Command) (config.Params, error) {
    // NO ARGS then loading conf from env variables
    if (len(args) == 0) {
        logDataDog = logger.NewLogger(false)
        return getConfFromEnv()
    }
    logDataDog = logger.NewLogger(true)
    // At least one arg then loading conf as parameters, must have 4
    if (len(args) != 4) {
        logDataDog.Error("ERROR: Invalid argument's number, must have 4, brokers url seprated by comma, number of kafka consumers, relativePath and topic")
        return config.Params{}, errors.New("Invalid arguments number");
    }
    brokers := args[0]
    nb, err := strconv.ParseInt(args[1], 10, 16)
    if err != nil {
        logDataDog.Error("ERROR: Invalid consumer number argument:", err)
        return config.Params{}, errors.New("Invalid consumer number argument")
    }
    nbConsumer := int(nb)
    if (nbConsumer > 20) {
        logDataDog.Error("ERROR: Too many consumers, max 20")
        return config.Params{}, errors.New("Too many consumers");
    }
    relativePath := args[2]
    topic := args[3]
    params := config.Params{MarkOnErrorFlag: false, MarkOnNotFoundFlag: false,
       BrokersUrls: brokers, ConsumersNumber: nbConsumer, RelativePath: relativePath, Topic: topic, AsCLI: true}
    mstatus, _ := cmd.Flags().GetBool("markOnError")
     if mstatus {
        params.MarkOnErrorFlag  = true
     }
     ntstatus, _ := cmd.Flags().GetBool("markOnNotFound")
     if ntstatus {
        params.MarkOnNotFoundFlag = true
    }
    return params, nil
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "rocket-storagemanager",
	Short: "Rocket Storage Manager executes filesystem commands and stornext commands",
	Long: `Rocket Storage Manager reads kafka workflow end events in order to cleanse the filesystem storage`,

	Run: func(cmd *cobra.Command, args []string) {
	    params, err := getArgsAndFlags(args, cmd);
	    if err != nil {
	        os.Exit(1)
	    }
	    logDataDog.Info("STARTING with config, brokers urls: ", params.BrokersUrls,
	        " ,Consumer numbers: ", params.ConsumersNumber,
	        " ,Selected workspace relativePath: ", params.RelativePath,
	        " ,topic: ", params.Topic,
	        " ,Run as a Service: ", !params.AsCLI,
	        " ,Marking message on error: ", params.MarkOnErrorFlag,
	        " ,Marking message on not found: ", params.MarkOnNotFoundFlag)
		ctx, cancel := context.WithCancel(context.Background())
		readChan := make(chan *kafka.ReadModel, 30)
		defer close(readChan)
		doneConsumeChan := make(chan bool)
		defer close(doneConsumeChan)
		finalDoneChan := make(chan bool)
        defer close(finalDoneChan)

		consumer := kafka.NewConsumer(kafkaConfig(), strings.Split(params.BrokersUrls, ","), readChan, logDataDog)

		for i := 0; i < params.ConsumersNumber; i++ {
			go consumer.Consume(ctx, group, []string{params.Topic}, doneConsumeChan)
		}
		go filesystem.RunInstance(readChan, doneConsumeChan, finalDoneChan, &params, logDataDog)

		// Listen system signal to stop goroutines by context
		sigc := make(chan os.Signal, 1)
		signal.Notify(sigc,
			syscall.SIGHUP,
			syscall.SIGINT,
			syscall.SIGTERM,
			syscall.SIGQUIT,
		)
		defer signal.Stop(sigc)
		select {
		    case <-sigc:
			    for i := 0; i < params.ConsumersNumber; i++ {
			        cancel()
			    }
		}
		<-finalDoneChan
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
    rootCmd.Flags().BoolP("markOnError", "e", false, "Mark message on errors except for the not found error")
    rootCmd.Flags().BoolP("markOnNotFound", "n", false, "Mark message on not found errors")
}

func kafkaConfig() *sarama.Config {
	sarama.Logger = log.New()

	version, err := sarama.ParseKafkaVersion(version)
	if err != nil {
		logDataDog.Error("Error parsing Kafka version: %v", err)
	}

	config := sarama.NewConfig()
	config.Consumer.Offsets.AutoCommit.Enable = true
	config.Version = version

	switch assignor {
	case "sticky":
		config.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategySticky
	case "roundrobin":
		config.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRoundRobin
	case "range":
		config.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRange
	default:
		logDataDog.Error("Unrecognized consumer group partition assignor: %s", assignor)
	}

	if oldest {
		config.Consumer.Offsets.Initial = sarama.OffsetOldest
	}

	return config
}
