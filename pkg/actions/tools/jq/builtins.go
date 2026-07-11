package jq

import (
	"github.com/carapace-sh/carapace"
)

// ActionBuiltins completes jq builtin function names.
func ActionBuiltins() carapace.Action {
	return carapace.ActionCallback(func(c carapace.Context) carapace.Action {
		// Zero-argument builtins (with () suffix)
		zeroArg := carapace.ActionValuesDescribed(
			"empty", "Produces no output",
			"error", "Produces an error with the input as message",
			"halt", "Stops the program with exit code 0",
			"halt_error", "Stops the program with input as error message",
			"infinite", "Produces positive infinity",
			"input", "Outputs one new input",
			"inputs", "Outputs all remaining inputs",
			"isnan", "True if input is NaN",
			"isinfinite", "True if input is infinite",
			"isfinite", "True if input is finite",
			"isnormal", "True if input is a normal number",
			"nan", "Produces NaN",
			"type", "Returns the type of input as a string",
			"builtins", "Lists all builtin functions",
			"input_filename", "Returns the name of the current input file",
			"input_line_number", "Returns the current line number",
			"length", "Returns the length of the input",
			"utf8bytelength", "Returns the byte length of a string",
			"keys", "Returns sorted keys of an object or indices of an array",
			"keys_unsorted", "Returns keys preserving insertion order",
			"to_entries", "Converts object to array of key-value pairs",
			"from_entries", "Converts array of key-value pairs to object",
			"ascii_downcase", "Converts ASCII letters to lowercase",
			"ascii_upcase", "Converts ASCII letters to uppercase",
			"explode", "Converts string to array of codepoints",
			"implode", "Converts array of codepoints to string",
			"tostring", "Converts to string representation",
			"tonumber", "Parses a number from a string",
			"tojson", "Serializes as JSON text",
			"fromjson", "Parses a JSON string",
			"add", "Reduces array with +",
			"reverse", "Reverses an array or string",
			"sort", "Sorts an array",
			"unique", "Removes duplicates from sorted array",
			"min", "Returns minimum element",
			"max", "Returns maximum element",
			"not", "Negates truthiness",
			"env", "Object of environment variables",
			"paths", "Outputs all paths in the input",
			"leaf_paths", "Outputs all paths to leaf values",
			"debug", "Writes debug info to stderr, passes input through",
			"stderr", "Outputs to stderr",
			"getpath", "Gets values at paths",
			"path", "Outputs paths matched by expression",
			"del", "Deletes values at paths",
			"delpaths", "Deletes multiple paths",
			"pick", "Projects specified paths",
		).Suffix("()").UidF(Uid("builtin", "args", "false"))

		// One-or-more-argument builtins (with ( suffix)
		withArgs := carapace.ActionValuesDescribed(
			"map", "Apply filter to each element: [.[] | f]",
			"map_values", "Apply filter to each value: .[] |= f",
			"select", "Pass input through if filter is true",
			"has", "Check if object has key or array has index",
			"in", "Check if input is a key in the given object",
			"contains", "Check if input contains the given value",
			"inside", "Check if input is inside the given value",
			"startswith", "Check if string starts with prefix",
			"endswith", "Check if string ends with suffix",
			"ltrimstr", "Remove prefix if present",
			"rtrimstr", "Remove suffix if present",
			"split", "Split string by separator",
			"join", "Join array with separator",
			"test", "Test if string matches regex",
			"match", "Match regex and return capture groups",
			"capture", "Match regex and bind named groups",
			"scan", "Find all matches of regex",
			"sub", "Substitute first match of regex",
			"gsub", "Substitute all matches of regex",
			"splits", "Split string by regex",
			"sort_by", "Sort array by given filter",
			"group_by", "Group array by given filter",
			"min_by", "Return element with minimum value of filter",
			"max_by", "Return element with maximum value of filter",
			"unique_by", "Remove duplicates by given filter",
			"any", "True if any element satisfies the filter",
			"all", "True if all elements satisfy the filter",
			"range", "Generate a range of numbers",
			"floor", "Round down to nearest integer",
			"sqrt", "Square root",
			"fabs", "Floating-point absolute value",
			"round", "Round to nearest integer",
			"getpath", "Get values at given path array",
			"setpath", "Set value at given path",
			"paths", "Output paths matching filter",
			"index", "Find first index of value",
			"rindex", "Find last index of value",
			"indices", "Find all indices of value",
			"nth", "Get nth element of a generator or array",
			"first", "Get first output of generator, or first of array",
			"last", "Get last output of generator, or last of array",
			"limit", "Take first n outputs of a generator",
			"skip", "Skip first n outputs of a generator",
			"isempty", "True if generator produces no output",
			"repeat", "Repeat expression indefinitely",
			"while", "Repeat while condition is true",
			"until", "Repeat until condition is true",
			"recurse", "Recursively apply filter",
			"walk", "Apply filter to every value recursively",
			"flatten", "Flatten nested arrays",
			"transpose", "Transpose a matrix",
			"bsearch", "Binary search in a sorted array",
			"ascii", "Convert character to codepoint",
			"combinations", "Cartesian product of generators",
			"ltrimstr", "Remove prefix from string",
			"strftime", "Format time with strftime",
			"strflocaltime", "Format time in local timezone",
			"strptime", "Parse time string",
			"mktime", "Convert broken-down time to epoch",
			"gmtime", "Convert epoch to broken-down time (UTC)",
			"localtime", "Convert epoch to broken-down time (local)",
			"now", "Current epoch time",
			"fromdate", "Parse ISO 8601 date",
			"todate", "Format as ISO 8601 date",
			"fromdateiso8601", "Parse ISO 8601 date",
			"todateiso8601", "Format as ISO 8601 date",
			"INDEX", "Index array by key (SQL-style)",
			"JOIN", "Join two arrays (SQL-style)",
			"IN", "Check if input is in a set (SQL-style)",
			"getpath", "Get value at path",
			"with_entries", "Apply filter to each entry: to_entries | map(f) | from_entries",
			"abs", "Absolute value",
		).Suffix("(").UidF(Uid("builtin", "args", "true"))

		return carapace.Batch(zeroArg, withArgs).ToA()
	}).Tag("builtins")
}

// ActionKeywords completes jq keywords that start expressions.
func ActionKeywords() carapace.Action {
	return carapace.ActionValuesDescribed(
		"if", "Conditional expression",
		"try", "Error handling",
		"reduce", "Reduce a generator into a single value",
		"foreach", "Iterate with intermediate results",
		"def", "Define a function",
		"label", "Define a label for break",
		"break", "Break out of a labeled expression",
	).Tag("keywords").UidF(Uid("keyword"))
}

// ActionKeywordTokens completes jq keyword tokens that appear within
// constructs (then, elif, else, end, catch, as).
func ActionKeywordTokens() carapace.Action {
	return carapace.ActionValuesDescribed(
		"then", "Then branch of if expression",
		"elif", "Else-if branch",
		"else", "Else branch",
		"end", "End of if expression",
		"catch", "Error handler for try",
		"as", "Variable binding",
	).Tag("keyword tokens").UidF(Uid("keyword-token")).NoSpace()
}

// ActionLiterals completes jq literal values.
func ActionLiterals() carapace.Action {
	return carapace.ActionValuesDescribed(
		"true", "Boolean true",
		"false", "Boolean false",
		"null", "Null value",
	).Tag("literals").UidF(Uid("literal"))
}

// ActionSpecialFilters completes special filter values like . and ..
func ActionSpecialFilters() carapace.Action {
	return carapace.ActionValuesDescribed(
		".", "Identity",
		"..", "Recursive descent",
	).Tag("special filters").UidF(Uid("special-filter"))
}

// ActionFormatStrings completes @format names.
func ActionFormatStrings() carapace.Action {
	return carapace.ActionValuesDescribed(
		"@text", "Convert to string with tostring",
		"@json", "Serialize as JSON",
		"@html", "HTML entity escaping",
		"@uri", "Percent-encode for URIs",
		"@csv", "CSV row formatting",
		"@tsv", "TSV row formatting",
		"@sh", "POSIX shell quoting",
		"@base64", "Base64 encode",
		"@base64d", "Base64 decode",
	).Tag("format strings").UidF(Uid("format-string"))
}

// ActionOperators completes operators valid at a given position.
func ActionOperators(ops []jqValidOperator) carapace.Action {
	return carapace.ActionCallback(func(c carapace.Context) carapace.Action {
		pairs := make([][2]string, 0, len(ops))
		for _, op := range ops {
			pairs = append(pairs, [2]string{op.Op, op.Description})
		}
		values := make([]string, 0, len(pairs)*2)
		for _, p := range pairs {
			values = append(values, p[0], p[1])
		}
		return carapace.ActionValuesDescribed(values...).UidF(Uid("operator"))
	})
}

// jqValidOperator is a local copy of jq.ValidOperator for the action layer.
type jqValidOperator struct {
	Op          string
	Description string
}
