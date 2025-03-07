package docstrings

const (
	identityIds = `A set of User Managed Identity ID's which should be assigned to the azure resource.`
)

// IdentityIds returns the docstring for the identity type attribute.
func IdentityIds() string {
	return identityIds
}
