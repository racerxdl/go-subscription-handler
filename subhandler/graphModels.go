package subhandler

import "encoding/json"

type graphQLPayload struct {
	Variables     map[string]interface{} `json:"variables"`
	OperationName string                 `json:"operationName"`
	Extensions    interface{}            `json:"extensions"`
	Query         string                 `json:"query"`
}

type graphQLWSRequest struct {
	Id      string         `json:"id"`
	Type    string         `json:"type"`
	Payload graphQLPayload `json:"payload"`
}

type graphQLWSMessage struct {
	Id      string      `json:"id"`
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

func (m graphQLWSMessage) ToGraphQLRequest() (*graphQLWSRequest, error) {
	q := &graphQLWSRequest{
		Type: m.Type,
		Id:   m.Id,
	}

	d, err := json.Marshal(m.Payload)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(d, &q.Payload)
	if err != nil {
		return nil, err
	}

	return q, nil
}
