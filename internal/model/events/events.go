package events

import (
    "github.com/sirupsen/logrus"
    "rocket-storagemanager/internal/utils"
)
var log = logger.NewLogger(false)

type WorkflowSuccess struct {
	Status          string  `json:"status"`
	WorkflowId      string  `json:"workflowId"`
	WorkflowType    string  `json:"workflowType"`
}

type WorkflowFailure struct {
	Status          string  `json:"status"`
	WorkflowId      string  `json:"workflowId"`
	WorkflowType    string  `json:"workflowType"`
	JobId           string  `json:"jobId"`
	Error           string  `json:"error"`
}

func (workflow *WorkflowSuccess) DataDogLogError(message string) {
	log.WithFields(logrus.Fields{
	    "workflowStatus":   workflow.Status,
		"workflowId":       workflow.WorkflowId,
		"workflowType":     workflow.WorkflowType,
	}).Error(message)
}
func (workflow *WorkflowSuccess) DataDogLogWarning(message string) {
	log.WithFields(logrus.Fields{
	    "workflowStatus":   workflow.Status,
		"workflowId":       workflow.WorkflowId,
		"workflowType":     workflow.WorkflowType,
	}).Warning(message)
}
func (workflow *WorkflowSuccess) DataDogLog(message string) {
	log.WithFields(logrus.Fields{
	    "workflowStatus":   workflow.Status,
		"workflowId":       workflow.WorkflowId,
		"workflowType":     workflow.WorkflowType,
	}).Info(message)
}

func (workflow *WorkflowFailure) DataDogLogError(message string) {
	log.WithFields(logrus.Fields{
	    "workflowStatus":   workflow.Status,
		"workflowId":       workflow.WorkflowId,
	    "jobId":            workflow.JobId,
		"workflowType":     workflow.WorkflowType,
		"workflowError":    workflow.Error,
	}).Error(message)
}
func (workflow *WorkflowFailure) DataDogLogWarning(message string) {
	log.WithFields(logrus.Fields{
	    "workflowStatus":   workflow.Status,
		"workflowId":       workflow.WorkflowId,
	    "jobId":            workflow.JobId,
		"workflowType":     workflow.WorkflowType,
		"workflowError":    workflow.Error,
	}).Warning(message)
}
func (workflow *WorkflowFailure) DataDogLog(message string) {
	log.WithFields(logrus.Fields{
	    "workflowStatus":   workflow.Status,
		"workflowId":       workflow.WorkflowId,
	    "jobId":            workflow.JobId,
		"workflowType":     workflow.WorkflowType,
		"WorkflowError":    workflow.Error,
	}).Info(message)
}
