package enum

type InventoryStatus string

const (
	InventoryDraft      InventoryStatus = "DRAFT"
	InventoryInProgress InventoryStatus = "IN_PROGRESS"
	InventoryCompleted  InventoryStatus = "COMPLETED"
)

func (InventoryStatus) Values() []string {
	return []string{
		string(InventoryDraft),
		string(InventoryInProgress),
		string(InventoryCompleted),
	}
}

func (s InventoryStatus) Value() string {
	return string(s)
}

func (s InventoryStatus) IsValid() bool {
	switch s {
	case InventoryDraft, InventoryInProgress, InventoryCompleted:
		return true
	}
	return false
}
