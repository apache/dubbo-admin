package query

import "dubbo-admin-ai/security"

func modelInput(value string) string {
	return security.DefaultSanitizer().ModelInput(value)
}
