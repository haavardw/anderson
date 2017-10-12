package anderson

import (
	"fmt"
	"log"
	"path"
	"path/filepath"
	"strings"

	"github.com/ryanuber/go-license"
)

const (
	maxParentHops = 10
)

type LicenseClassifier struct {
	Config Config
}

func (c LicenseClassifier) vendorPath(p string) string {
	vendor := c.parentPath(p, 3)
	if path.Base(vendor) != "vendor" {
		return ""
	}
	repo := path.Base(c.parentPath(p, 2))
	p0 := path.Base(c.parentPath(p, 1))
	p1 := path.Base(c.parentPath(p, 0))

	return fmt.Sprintf("%s/%s_%s_%s", p, repo, p0, p1)
}

func (c LicenseClassifier) Classify(path string, importPath string) (LicenseStatus, string, string, error) {
	for hops := 0; hops < maxParentHops; hops++ {
		newPath := c.parentPath(path, hops)
		if c.pathIsAGopath(newPath) {
			break
		}

		status, licenseType, err := c.classifyPath(newPath, importPath)

		if status != LicenseTypeNoLicense {
			return status, newPath, licenseType, err
		}

		newVendorPath := c.vendorPath(newPath)
		if newVendorPath != "" {
			log.Println("TESTING VENDORPATH", newVendorPath)
			status, licenseType, err = c.classifyPath(newVendorPath, importPath)

			if status != LicenseTypeNoLicense {
				return status, newPath, licenseType, err
			}
		}
	}

	return LicenseTypeNoLicense, path, "Unknown", nil
}

func (c LicenseClassifier) classifyPath(path string, importPath string) (LicenseStatus, string, error) {
	l, err := license.NewFromDir(path)

	if err != nil {
		switch err.Error() {
		case license.ErrNoLicenseFile:
			if contains(c.Config.Exceptions, importPath) {
				return LicenseTypeAllowed, "Unknown", nil
			}

			return LicenseTypeNoLicense, "Unknown", nil
		case license.ErrUnrecognizedLicense:
			if contains(c.Config.Exceptions, importPath) {
				return LicenseTypeAllowed, "Unknown", nil
			}

			return LicenseTypeUnknown, "Unknown", nil
		default:
			return LicenseTypeUnknown, "Error", fmt.Errorf("Could not determine license for: %s", importPath)
		}
	}

	if contains(c.Config.Blacklist, l.Type) {
		return LicenseTypeBanned, l.Type, nil
	}

	if contains(c.Config.Whitelist, l.Type) {
		return LicenseTypeAllowed, l.Type, nil
	}

	if contains(c.Config.Exceptions, importPath) {
		return LicenseTypeAllowed, l.Type, nil
	}

	return LicenseTypeMarginal, l.Type, nil
}

func (c LicenseClassifier) pathIsAGopath(path string) bool {
	paths, _ := Gopaths()

	return contains(paths, path)
}

func (c LicenseClassifier) parentPath(path string, hops int) string {
	dots := strings.Fields(strings.Repeat(".. ", hops))
	elements := []string{}
	elements = append(elements, path)
	elements = append(elements, dots...)

	return filepath.Clean(filepath.Join(elements...))
}

func contains(haystack []string, needle string) bool {
	for _, element := range haystack {
		if element == needle {
			return true
		}
	}
	return false
}
