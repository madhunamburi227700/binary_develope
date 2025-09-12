package utils

import (
    "github.com/rs/zerolog/log"
)

type ErrorLogger struct {
    Component string
}

func NewErrorLogger(component string) *ErrorLogger {
    return &ErrorLogger{Component: component}
}

func (el *ErrorLogger) LogError(err error, message string, fields map[string]interface{}) {
    logEvent := log.Error().Err(err).Str("component", el.Component).Str("message", message)
    
    for key, value := range fields {
        logEvent = logEvent.Interface(key, value)
    }
    
    logEvent.Msg("Error occurred")
}

func (el *ErrorLogger) LogWarning(message string, fields map[string]interface{}) {
    logEvent := log.Warn().Str("component", el.Component).Str("message", message)
    
    for key, value := range fields {
        logEvent = logEvent.Interface(key, value)
    }
    
    logEvent.Msg("Warning")
}

func (el *ErrorLogger) LogInfo(message string, fields map[string]interface{}) {
    logEvent := log.Info().Str("component", el.Component).Str("message", message)
    
    for key, value := range fields {
        logEvent = logEvent.Interface(key, value)
    }
    
    logEvent.Msg("Info")
}