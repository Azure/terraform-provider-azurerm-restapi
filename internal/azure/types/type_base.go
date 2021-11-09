package types

type TypeBase interface {
	AsTypeBase() *TypeBase
	Validate(interface{}, string) []error
}
