package handlers

type ResponseEnvelope struct {
	ApiVersion     uint8            `json:"api_version"`
	ConnectionId   string           `json:"connection_id"`
	Info           string           `json:"info"`
	RowsCount      uint32           `json:"rows_count"`
	ExceedsMaxRows bool             `json:"exceeds_max_rows"`
	Rows           []map[string]any `json:"rows"`
}
