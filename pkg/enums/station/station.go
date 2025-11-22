package station

import "strings"

type Station struct {
	Name string
}

func (s Station) Code() string {
	return s.Name
}

func (s Station) Label() string {
	// Capitalize first letter
	if len(s.Name) == 0 {
		return ""
	}
	return strings.ToUpper(s.Name[:1]) + s.Name[1:]
}

type Enum struct {
	Kitchen Station
	Dessert Station
	Bar     Station
	Coffee  Station
	Other   Station
}

var Stations = Enum{
	Kitchen: Station{Name: "kitchen"},
	Dessert: Station{Name: "dessert"},
	Bar:     Station{Name: "bar"},
	Coffee:  Station{Name: "coffee"},
	Other:   Station{Name: "other"},
}

var All = []Station{
	Stations.Kitchen,
	Stations.Dessert,
	Stations.Bar,
	Stations.Coffee,
	Stations.Other,
}

// ByName returns the station for a given name, or nil if not found
func ByName(name string) *Station {
	for _, s := range All {
		if s.Name == name {
			return &s
		}
	}
	return nil
}
