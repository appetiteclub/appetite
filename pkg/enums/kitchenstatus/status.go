package kitchenstatus

import (
	"strings"
)

type Status struct {
	Name string
}

func (s Status) Code() string {
	return s.Name
}

func (s Status) Label() string {
	parts := strings.Split(s.Name, "-")
	for i := range parts {
		if len(parts[i]) > 0 {
			parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	return strings.Join(parts, " ")
}

type Enum struct {
	Created   Status
	Accepted  Status
	Started   Status
	Ready     Status
	Delivered Status
	Reject    Status
	Standby   Status
	Block     Status
	Cancelled Status
}

var Statuses = Enum{
	Created:   Status{Name: "created"},
	Accepted:  Status{Name: "accepted"},
	Started:   Status{Name: "started"},
	Ready:     Status{Name: "ready"},
	Delivered: Status{Name: "delivered"},
	Reject:    Status{Name: "reject"},
	Standby:   Status{Name: "standby"},
	Block:     Status{Name: "block"},
	Cancelled: Status{Name: "cancelled"},
}

var All = []Status{
	Statuses.Created,
	Statuses.Accepted,
	Statuses.Started,
	Statuses.Ready,
	Statuses.Delivered,
	Statuses.Reject,
	Statuses.Standby,
	Statuses.Block,
	Statuses.Cancelled,
}

// ByName returns the status for a given name, or nil if not found
func ByName(name string) *Status {
	for _, s := range All {
		if s.Name == name {
			return &s
		}
	}
	return nil
}
