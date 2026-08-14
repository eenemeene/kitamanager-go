package i18n

// The registry maps an English message to the catalogue entry that localizes it.
//
// Call sites keep writing English format strings — `apperror.BadRequest("child
// %d not found in this organization", id)` — because that is what makes an error
// readable where it is raised, and what keeps `Error()` English for the logs.
// This table is the bridge from that string to a stable message ID.
//
// Why a bridge instead of putting IDs at the call sites: the English string is a
// poor translation key, because rewording it silently orphans the translation,
// but it is an excellent thing to read in code. The registry gets both — the
// call site stays prose, the catalogue gets an ID that survives rewording, and
// TestRegistryMatchesSource fails if a message is reworded without updating the
// entry, which is the failure that would otherwise be silent.
//
// Only messages worth localizing need an entry. Anything absent renders in
// English, which is the correct outcome for the internal-failure messages that
// make up most of the tree.

// entry describes how one English message becomes a localized one.
type entry struct {
	// ID is the catalogue key. Stable across rewordings of the English text.
	ID string
	// Args names the format's verbs, in order, so the catalogue can refer to
	// them by name. A translator writing German needs to move them around the
	// sentence, and "{{.Count}}" can be moved where "%d" cannot.
	Args []string
	// Plural is the name of the argument that selects the plural form, empty if
	// the message does not pluralize. go-i18n applies CLDR rules per language,
	// so German gets German's rules rather than English's.
	Plural string
}

// registry is keyed by the exact English format string a call site passes.
var registry = map[string]entry{
	"child %d not found in this organization": {
		ID:   "child.not_found_in_organization",
		Args: []string{"ChildID"},
	},
	"section %d not found in this organization": {
		ID:   "section.not_found_in_organization",
		Args: []string{"SectionID"},
	},
	// This one is why the plural support is not theoretical: the English source
	// reads "with 1 currently-assigned children" today, which is wrong before
	// any translation is involved.
	"cannot delete section with %d currently-assigned children; reassign them first": {
		ID:     "section.delete.has_children",
		Args:   []string{"Count"},
		Plural: "Count",
	},
	"cannot delete section with %d currently-assigned employees; reassign them first": {
		ID:     "section.delete.has_employees",
		Args:   []string{"Count"},
		Plural: "Count",
	},
}

// titleID returns the catalogue ID for an error code's title. Derived rather
// than tabulated: the code is already a stable slug, so a new code gets a
// predictable ID and there is no second list to forget to update.
func titleID(code string) string {
	return "error.title." + code
}
