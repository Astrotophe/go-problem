# Rocket Storage Manager

Service which will read a kafka topic, this topic contains the workflows end events. StorageManager, after each workflow
end
will cleanse the _/workspace directory, it will search for every directory whoose name is the workflowId and deletes it.

There is 2 execution's type, run as a CLI or as a Service, to run the binary as a service just execute it without any
argument, don't forget to update your env variables

#### ENV VARIABLES to implement

KAFKA_BROKERS_SERVERS
KAFKA_CONSUMER_NUMBER
ROCKET_WORKSPACE_PATH
KAFKA_TOPIC_NAME
ROCKET_LOG_PATH

go build rocket-storagemanager

./rocket-storagemanager brockers-urls kafkaConsumerNumber relativePath topic --Flags

## flags

false per default

--markOnError / -e : mark kafka message as consumed when an error is met, like an invalid format of the message.
This flag ignores NotFound errors.

--markOnNotFound / -n : mark kefka message as consumed when the not found error is met.

### example

#### as cli

./rocket-storagemanager "dev02-hbm-kfb01.aws.cpdev.local:9092,dev02-hbm-kfb02.aws.cpdev.local:
9092,dev02-hbm-kfb03.aws.cpdev.local:9092" 3 "/Users/test/workspace/" "workflow-results" -n

#### as a service

./rocket-storagemanager
