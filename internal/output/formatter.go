package output

import (
	"encoding/json"
	"os"
)

var JSONMode bool

func SetJSONMode(enabled bool) {
	JSONMode = enabled
}

func IsJSONMode() bool {
	return JSONMode
}

func PrintJSON(data interface{}) error {
	envelope := Envelope{
		Version: APIVersion,
		Data:    data,
	}
	return writeJSON(envelope)
}

func PrintError(code, message string) error {
	envelope := ErrorEnvelope{
		Version: APIVersion,
		Error: &ErrorInfo{
			Code:    code,
			Message: message,
		},
	}
	return writeJSON(envelope)
}

func writeJSON(v interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(v)
}

func FormatMachineList(machines []MachineJSON) error {
	if !JSONMode {
		return nil
	}
	data := &MachineListData{
		Machines: machines,
	}
	return PrintJSON(data)
}
