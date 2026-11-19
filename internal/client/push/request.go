package push

type Request struct {
	RequestID   string      `json:"request_id"`
	RecipientID string      `json:"recipient_id"`
	Kind        string      `json:"kind"`
	Data        RequestData `json:"data"`
}

type RequestData struct {
	TripID string `json:"trip_id"`
}

const KindRequestPosition = "REQUEST_POSITION"
