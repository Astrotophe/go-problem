package filesystem

import (
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io/fs"
	"os"
	"path/filepath"
	"rocket-storagemanager/internal/kafka"
	"rocket-storagemanager/internal/model/config"
	"rocket-storagemanager/internal/model/events"
	"time"
)

var logDataDog *log.Logger

type ErrorType string

const (
	NoError       ErrorType = "NoError"
	NotFound                = "NotFound"
	NoAccess                = "NoAccess"
	InvalidFormat           = "InvalidFormat"
	GeneralError            = "GeneralError"
)

func findDirectories(workflowId string, relativeWorkspacePath string) []string {
	var dirsToDelete []string
	time.Sleep(8 * time.Second)
	filepath.WalkDir(relativeWorkspacePath,
		func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() && d.Name() == workflowId {
				logDataDog.Info("Path found: ", path, "for workflow ID: ", workflowId)
				dirsToDelete = append(dirsToDelete, path)
			}
			return nil
		})
	return dirsToDelete
}

func deleteDirectory(directoryPath string, workflowId string) ErrorType {
	if _, err := os.Stat(directoryPath); err != nil {
		if os.IsNotExist(err) {
			logDataDog.WithFields(log.Fields{
				"context":    "deleting-directory",
				"workflowId": workflowId,
			}).Info(fmt.Sprintf("INFO: Directory doesnt exists: ", directoryPath))
			return NotFound
		}
		return GeneralError
	}
	errRemove := os.RemoveAll(directoryPath)
	if errRemove != nil {
		if os.IsPermission(errRemove) {
			logDataDog.WithFields(log.Fields{
				"context":    "deleting-directory",
				"workflowId": workflowId,
			}).Error(fmt.Sprintf("ERROR: Service doesnt have rights to access: ", directoryPath))
			return NoAccess
		}
		return GeneralError
	}
	return NoError
}

func deleteDirectories(dirs []string, workflowId string) ErrorType {
	var allDeleted []ErrorType
	for _, directoryPath := range dirs {
		allDeleted = append(allDeleted, deleteDirectory(directoryPath, workflowId))
	}
	for _, x := range allDeleted {
		if x != NoError {
			return x
		}
	}
	return NoError
}

// TODO add workflowEvent parent model and separate logic of parsing message and cleaning directory
func resolveMessage(message string, relativePath string) ErrorType {
	// json decode
	var msgValue map[string]interface{}
	if err := json.Unmarshal([]byte(message), &msgValue); err != nil {
		logDataDog.Error("ERROR: Something went wrong in parsing received message :", msgValue)
		return InvalidFormat
	}
	if _, statusExists := msgValue["status"]; !statusExists {
		logDataDog.Error("ERROR: received message has incorrect format: ", message)
		return InvalidFormat
	}
	switch status := msgValue["status"].(string); status {
	case "success":
		success := events.WorkflowSuccess{}
		err := json.Unmarshal([]byte(message), &success)
		if err != nil {
			logDataDog.Error("ERROR: while formating received message for success model: ", err)
			return InvalidFormat
		}
		success.DataDogLog("Received SUCCESS workflow event")
		directoriesToDelete := findDirectories(success.WorkflowId, relativePath)
		if len(directoriesToDelete) < 1 {
			success.DataDogLog(fmt.Sprintf("INFO: no directory has been found for workflowId: ", success.WorkflowId))
			return NotFound
		}
		errType := deleteDirectories(directoriesToDelete, success.WorkflowId)
		return errType
	case "failure":
		failure := events.WorkflowFailure{}
		err := json.Unmarshal([]byte(message), &failure)
		if err != nil {
			logDataDog.Error("ERROR: while formating received message for failure model: ", err)
			return InvalidFormat
		}
		failure.DataDogLog("Received FAILURE workflow event")
		directoriesToDelete := findDirectories(failure.WorkflowId, relativePath)
		if len(directoriesToDelete) < 1 {
			failure.DataDogLog(fmt.Sprintf("INFO: no directory has been found for workflowId: ", failure.WorkflowId))
			return NotFound
		}
		errType := deleteDirectories(directoriesToDelete, failure.WorkflowId)
		return errType
	default:
		logDataDog.Error("ERROR: Incorrect topic message status:", status)
		return InvalidFormat

	}
}

func cleanWorkspace(data *kafka.ReadModel, params *config.Params) {
	errType := resolveMessage(string(data.Message.Value), params.RelativePath)
	switch errType {
	case NoError:
		data.Session.MarkMessage(data.Message, "")
		logDataDog.Info("EVENT MARKED for message:", string(data.Message.Value))
	case NotFound:
		if params.MarkOnNotFoundFlag == true {
			data.Session.MarkMessage(data.Message, "")
			logDataDog.Info("EVENT MARKED for NotFoundError because flag -n is true, message:", string(data.Message.Value))
		} else {
			logDataDog.Info("WARNING: event UNMARKED for message:", string(data.Message.Value))
		}
	case NoAccess, InvalidFormat, GeneralError:
		if params.MarkOnErrorFlag == true {
			data.Session.MarkMessage(data.Message, "")
			logDataDog.Info("EVENT MARKED for Error because flag -e is true, message:", string(data.Message.Value))
		} else {
			logDataDog.Info("WARNING: event UNMARKED for message:", string(data.Message.Value))
		}
	}
}

func RunInstance(readChan chan *kafka.ReadModel, doneChan chan bool, finalDoneChan chan bool, params *config.Params, logg *log.Logger) {
	done := false
	consumersStooped := 0
	logDataDog = logg
	for !done {
		select {
		case data := <-readChan:
			go cleanWorkspace(data, params)
		case <-doneChan:
			consumersStooped++
			if consumersStooped >= params.ConsumersNumber {
				done = true
			}
		}
	}
	finalDoneChan <- true
}
