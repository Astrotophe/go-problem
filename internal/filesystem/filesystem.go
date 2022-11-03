package filesystem

import (
	"context"
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"io/fs"
	"os"
	"path/filepath"
	"rocket-storagemanager/internal/kafka"
	"rocket-storagemanager/internal/model/config"
	"rocket-storagemanager/internal/model/events"
	"sync"
	"time"
)

const (
	NoError       ErrorType = "NoError"
	NotFound                = "NotFound"
	NoAccess                = "NoAccess"
	InvalidFormat           = "InvalidFormat"
	GeneralError            = "GeneralError"
)

type fileSystem struct {
	logger *log.Logger
	params config.Params
}

type FileSystem interface {
	Run(context.Context, <-chan *kafka.ReadModel, *sync.WaitGroup)
}

type ErrorType string

func (f *fileSystem) findDirectories(workflowId string, relativeWorkspacePath string) ([]string, error) {
	var dirsToDelete []string
	time.Sleep(8 * time.Second)
	err := filepath.WalkDir(
		relativeWorkspacePath,
		func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() && d.Name() == workflowId {
				f.logger.Info("Path found: ", path, "for workflow ID: ", workflowId)
				dirsToDelete = append(dirsToDelete, path)
			}
			return nil
		},
	)
	if err != nil {
		return nil, err
	}
	return dirsToDelete, nil
}

func (f *fileSystem) deleteDirectory(directoryPath string, workflowId string) ErrorType {
	if _, err := os.Stat(directoryPath); err != nil {
		if os.IsNotExist(err) {
			f.logger.WithFields(log.Fields{
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
			f.logger.WithFields(log.Fields{
				"context":    "deleting-directory",
				"workflowId": workflowId,
			}).Error(fmt.Sprintf("ERROR: Service doesnt have rights to access: ", directoryPath))
			return NoAccess
		}
		return GeneralError
	}
	return NoError
}

func (f *fileSystem) deleteDirectories(dirs []string, workflowId string) ErrorType {
	var allDeleted []ErrorType
	for _, directoryPath := range dirs {
		allDeleted = append(allDeleted, f.deleteDirectory(directoryPath, workflowId))
	}
	for _, x := range allDeleted {
		if x != NoError {
			return x
		}
	}
	return NoError
}

// TODO add workflowEvent parent model and separate logic of parsing message and cleaning directory
func (f *fileSystem) resolveMessage(message string, relativePath string) ErrorType {
	// json decode
	var msgValue map[string]interface{}
	if err := json.Unmarshal([]byte(message), &msgValue); err != nil {
		f.logger.Error("ERROR: Something went wrong in parsing received message :", msgValue)
		return InvalidFormat
	}
	if _, statusExists := msgValue["status"]; !statusExists {
		f.logger.Error("ERROR: received message has incorrect format: ", message)
		return InvalidFormat
	}
	switch status := msgValue["status"].(string); status {
	case "success":
		success := events.WorkflowSuccess{}
		err := json.Unmarshal([]byte(message), &success)
		if err != nil {
			f.logger.Error("ERROR: while formating received message for success model: ", err)
			return InvalidFormat
		}
		success.DataDogLog("Received SUCCESS workflow event")
		directoriesToDelete, err := f.findDirectories(success.WorkflowId, relativePath)
		if err != nil {
			f.logger.WithError(err).Debug("Error on walk")
		}
		if len(directoriesToDelete) < 1 {
			success.DataDogLog(fmt.Sprintf("INFO: no directory has been found for workflowId: ", success.WorkflowId))
			return NotFound
		}
		errType := f.deleteDirectories(directoriesToDelete, success.WorkflowId)
		return errType
	case "failure":
		failure := events.WorkflowFailure{}
		err := json.Unmarshal([]byte(message), &failure)
		if err != nil {
			f.logger.Error("ERROR: while formating received message for failure model: ", err)
			return InvalidFormat
		}
		failure.DataDogLog("Received FAILURE workflow event")
		directoriesToDelete, err := f.findDirectories(failure.WorkflowId, relativePath)
		if err != nil {
			f.logger.WithError(err).Debug("Error on walk")
		}
		if len(directoriesToDelete) < 1 {
			failure.DataDogLog(fmt.Sprintf("INFO: no directory has been found for workflowId: ", failure.WorkflowId))
			return NotFound
		}
		errType := f.deleteDirectories(directoriesToDelete, failure.WorkflowId)
		return errType
	default:
		f.logger.Error("ERROR: Incorrect topic message status:", status)
		return InvalidFormat

	}
}

func (f *fileSystem) cleanWorkspace(data *kafka.ReadModel, params config.Params) {
	errType := f.resolveMessage(string(data.Message.Value), params.RelativePath)
	switch errType {
	case NoError:
		data.Session.MarkMessage(data.Message, "")
		f.logger.Info("EVENT MARKED for message:", string(data.Message.Value))
	case NotFound:
		if params.MarkOnNotFoundFlag == true {
			data.Session.MarkMessage(data.Message, "")
			f.logger.Info("EVENT MARKED for NotFoundError because flag -n is true, message:", string(data.Message.Value))
		} else {
			f.logger.Info("WARNING: event UNMARKED for message:", string(data.Message.Value))
		}
	case NoAccess, InvalidFormat, GeneralError:
		if params.MarkOnErrorFlag == true {
			data.Session.MarkMessage(data.Message, "")
			f.logger.Info("EVENT MARKED for Error because flag -e is true, message:", string(data.Message.Value))
		} else {
			f.logger.Info("WARNING: event UNMARKED for message:", string(data.Message.Value))
		}
	}
}

func (f *fileSystem) Run(ctx context.Context, readChan <-chan *kafka.ReadModel, wg *sync.WaitGroup) {
	for {
		select {
		case data := <-readChan:
			// Why you are using a goroutine here as it's totally useless and can generate too many threads and
			// a data race or/and race condition for a same directory. Let this be synchronous...
			f.cleanWorkspace(data, f.params)
		case <-ctx.Done():
			wg.Done()
			return
		}
	}
}

func New(params config.Params, logger *log.Logger) FileSystem {
	return &fileSystem{
		logger: logger,
		params: params,
	}
}
