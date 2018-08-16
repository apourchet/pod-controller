package controller

func intsContain(arr []int, needle int) bool {
	for _, i := range arr {
		if i == needle {
			return true
		}
	}
	return false
}

func filterErrors(errs []error) []error {
	filtered := []error{}
	for _, err := range errs {
		if err == nil {
			continue
		}
		filtered = append(filtered, err)
	}
	return filtered
}

func stringifyErrors(errs []error) []string {
	errs = filterErrors(errs)
	out := []string{}
	for _, err := range errs {
		out = append(out, err.Error())
	}
	return out
}
