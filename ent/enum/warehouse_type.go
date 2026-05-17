package enum

type WarehouseType string

const (
	Main    WarehouseType = "MAIN"
	Hall    WarehouseType = "HALL"
	Transit WarehouseType = "TRANSIT"
)

func (WarehouseType) Values() []string {
	return []string{
		string(Main),
		string(Hall),
		string(Transit),
	}
}

func (w WarehouseType) Value() string {
	return string(w)
}

func (w WarehouseType) IsValid() bool {
	switch w {
	case Main, Hall, Transit:
		return true
	}
	return false
}
