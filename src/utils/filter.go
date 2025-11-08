package utils

func FilterUniqueStrings(input []string) []string {
	seen := make(map[string]bool)
	result := []string{}

	for _, d := range input {
		if !seen[d] {
			seen[d] = true
			result = append(result, d)
		}
	}

	return result
}
