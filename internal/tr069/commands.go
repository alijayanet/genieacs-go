package tr069

import (
	"encoding/xml"
	"fmt"
	"regexp"
	"strings"
)

// CWMP RPC Commands

// GetParameterValues creates a GetParameterValues request
func CreateGetParameterValues(id string, parameterNames []string) []byte {
	params := ""
	for _, name := range parameterNames {
		params += fmt.Sprintf(`<string>%s</string>`, name)
	}

	return []byte(fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/" 
               xmlns:cwmp="urn:dslforum-org:cwmp-1-0"
               xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
               xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">%s</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:GetParameterValues>
      <ParameterNames soap-enc:arrayType="xsd:string[%d]">
        %s
      </ParameterNames>
    </cwmp:GetParameterValues>
  </soap:Body>
</soap:Envelope>`, id, len(parameterNames), params))
}

// SetParameterValues creates a SetParameterValues request
func CreateSetParameterValues(id string, params map[string]interface{}) []byte {
	paramList := ""
	for name, value := range params {
		paramList += fmt.Sprintf(`
        <ParameterValueStruct>
          <Name>%s</Name>
          <Value xsi:type="xsd:string">%v</Value>
        </ParameterValueStruct>`, name, value)
	}

	return []byte(fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/" 
               xmlns:cwmp="urn:dslforum-org:cwmp-1-0"
               xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
               xmlns:xsd="http://www.w3.org/2001/XMLSchema">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">%s</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:SetParameterValues>
      <ParameterList soap-enc:arrayType="cwmp:ParameterValueStruct[%d]">
        %s
      </ParameterList>
      <ParameterKey>goacs-%s</ParameterKey>
    </cwmp:SetParameterValues>
  </soap:Body>
</soap:Envelope>`, id, len(params), paramList, id))
}

// CreateReboot creates a Reboot request
func CreateReboot(id string, commandKey string) []byte {
	return []byte(fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/" 
               xmlns:cwmp="urn:dslforum-org:cwmp-1-0">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">%s</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:Reboot>
      <CommandKey>%s</CommandKey>
    </cwmp:Reboot>
  </soap:Body>
</soap:Envelope>`, id, commandKey))
}

// CreateFactoryReset creates a FactoryReset request
func CreateFactoryReset(id string) []byte {
	return []byte(fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/" 
               xmlns:cwmp="urn:dslforum-org:cwmp-1-0">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">%s</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:FactoryReset/>
  </soap:Body>
</soap:Envelope>`, id))
}

// CreateDownload creates a Download request for firmware update
func CreateDownload(id string, fileType string, url string, fileSize int64, username, password string) []byte {
	authInfo := ""
	if username != "" {
		authInfo = fmt.Sprintf(`
      <Username>%s</Username>
      <Password>%s</Password>`, username, password)
	}

	return []byte(fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/" 
               xmlns:cwmp="urn:dslforum-org:cwmp-1-0">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">%s</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:Download>
      <CommandKey>goacs-fw-%s</CommandKey>
      <FileType>%s</FileType>
      <URL>%s</URL>
      <FileSize>%d</FileSize>%s
      <TargetFileName></TargetFileName>
      <DelaySeconds>0</DelaySeconds>
      <SuccessURL></SuccessURL>
      <FailureURL></FailureURL>
    </cwmp:Download>
  </soap:Body>
</soap:Envelope>`, id, id, fileType, url, fileSize, authInfo))
}

// CreateGetRPCMethods creates a GetRPCMethods request
func CreateGetRPCMethods(id string) []byte {
	return []byte(fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/" 
               xmlns:cwmp="urn:dslforum-org:cwmp-1-0">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">%s</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:GetRPCMethods/>
  </soap:Body>
</soap:Envelope>`, id))
}

// CreateGetParameterNames creates a GetParameterNames request
func CreateGetParameterNames(id string, parameterPath string, nextLevel bool) []byte {
	next := "0"
	if nextLevel {
		next = "1"
	}
	return []byte(fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/" 
               xmlns:cwmp="urn:dslforum-org:cwmp-1-0">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">%s</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:GetParameterNames>
      <ParameterPath>%s</ParameterPath>
      <NextLevel>%s</NextLevel>
    </cwmp:GetParameterNames>
  </soap:Body>
</soap:Envelope>`, id, parameterPath, next))
}

// CreateAddObject creates an AddObject request
func CreateAddObject(id string, objectName string, parameterKey string) []byte {
	return []byte(fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/" 
               xmlns:cwmp="urn:dslforum-org:cwmp-1-0">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">%s</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:AddObject>
      <ObjectName>%s</ObjectName>
      <ParameterKey>%s</ParameterKey>
    </cwmp:AddObject>
  </soap:Body>
</soap:Envelope>`, id, objectName, parameterKey))
}

// CreateDeleteObject creates a DeleteObject request
func CreateDeleteObject(id string, objectName string, parameterKey string) []byte {
	return []byte(fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/" 
               xmlns:cwmp="urn:dslforum-org:cwmp-1-0">
  <soap:Header>
    <cwmp:ID soap:mustUnderstand="1">%s</cwmp:ID>
  </soap:Header>
  <soap:Body>
    <cwmp:DeleteObject>
      <ObjectName>%s</ObjectName>
      <ParameterKey>%s</ParameterKey>
    </cwmp:DeleteObject>
  </soap:Body>
</soap:Envelope>`, id, objectName, parameterKey))
}

// ============== Response Parsers ==============

// GetParameterValuesResponse represents the response from GetParameterValues
type GetParameterValuesResponse struct {
	ParameterList []ParsedParameterValue
}

// ParsedParameterValue represents a parsed parameter name/value pair from GetParameterValuesResponse
type ParsedParameterValue struct {
	Name  string
	Value string
	Type  string
}

// ParseGetParameterValuesResponse parses a GetParameterValuesResponse
func ParseGetParameterValuesResponse(body []byte) (*GetParameterValuesResponse, error) {
	response := &GetParameterValuesResponse{
		ParameterList: []ParsedParameterValue{},
	}

	bodyStr := string(body)

	// Find all ParameterValueStruct elements
	// Pattern: <ParameterValueStruct>...<Name>...</Name>...<Value...>...</Value>...</ParameterValueStruct>
	namePattern := regexp.MustCompile(`<Name>([^<]+)</Name>`)
	valuePattern := regexp.MustCompile(`<Value[^>]*>([^<]*)</Value>`)

	// Split by ParameterValueStruct
	parts := strings.Split(bodyStr, "<ParameterValueStruct>")
	for i, part := range parts {
		if i == 0 {
			continue // Skip first part before any ParameterValueStruct
		}

		nameMatch := namePattern.FindStringSubmatch(part)
		valueMatch := valuePattern.FindStringSubmatch(part)

		if len(nameMatch) >= 2 {
			param := ParsedParameterValue{
				Name: strings.TrimSpace(nameMatch[1]),
			}
			if len(valueMatch) >= 2 {
				param.Value = strings.TrimSpace(valueMatch[1])
			}
			response.ParameterList = append(response.ParameterList, param)
		}
	}

	return response, nil
}

// SetParameterValuesResponse represents the response from SetParameterValues
type SetParameterValuesResponse struct {
	Status int // 0 = applied, 1 = will apply after reboot
}

// FaultResponse represents a CWMP fault
type FaultResponse struct {
	FaultCode   string
	FaultString string
	Detail      struct {
		FaultCode   string
		FaultString string
	}
}

// Common CWMP Fault Codes
const (
	FaultMethodNotSupported          = "8000"
	FaultRequestDenied               = "8001"
	FaultInternalError               = "8002"
	FaultInvalidArguments            = "8003"
	FaultResourcesExceeded           = "8004"
	FaultInvalidParameterName        = "9005"
	FaultInvalidParameterType        = "9006"
	FaultInvalidParameterValue       = "9007"
	FaultParameterNotWritable        = "9008"
	FaultNotificationRequestRejected = "9009"
)

// XML Marshal helper for SOAP envelope
func MarshalSOAPEnvelope(header *SOAPHeader, body interface{}) ([]byte, error) {
	envelope := struct {
		XMLName xml.Name    `xml:"soap:Envelope"`
		XMLNS   string      `xml:"xmlns:soap,attr"`
		CWMP    string      `xml:"xmlns:cwmp,attr"`
		Header  *SOAPHeader `xml:"soap:Header,omitempty"`
		Body    interface{} `xml:"soap:Body"`
	}{
		XMLNS:  "http://schemas.xmlsoap.org/soap/envelope/",
		CWMP:   "urn:dslforum-org:cwmp-1-0",
		Header: header,
		Body:   body,
	}

	return xml.MarshalIndent(envelope, "", "  ")
}
