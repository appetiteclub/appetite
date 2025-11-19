package operations

import (
	"encoding/json"
	"errors"

	"github.com/aquamarinepk/aqm"
)

// decodeSuccessResponse copies the dynamic response payload into dest.
func decodeSuccessResponse(resp *aqm.SuccessResponse, dest interface{}) error {
	if resp == nil {
		return errors.New("nil success response")
	}

	raw, err := json.Marshal(resp.Data)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(raw, dest); err != nil {
		return err
	}

	return nil
}
