package enum

type MovementType string

const (
	Receipt  MovementType = "RECEIPT"
	Transfer MovementType = "TRANSFER"
	WriteOff MovementType = "WRITE_OFF"
	Gift     MovementType = "GIFT"
)

func (MovementType) Values() []string {
	return []string{
		string(Receipt),
		string(Transfer),
		string(WriteOff),
		string(Gift),
	}
}

func (m MovementType) Value() string {
	return string(m)
}

func (m MovementType) IsValid() bool {
	switch m {
	case Receipt, Transfer, WriteOff, Gift:
		return true
	}
	return false
}
